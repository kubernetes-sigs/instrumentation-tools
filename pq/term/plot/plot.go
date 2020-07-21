/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package plot

import (
	"math"
)

const (
	brailleCellWidth = 2
	brailleCellHeight = 4
	brailleCellPositions = brailleCellWidth*brailleCellHeight
)

// SeriesId identifies some series.  The zero value is reserved for unset.
type SeriesId uint16
const NoSeries = SeriesId(0)

type SeriesSet []Series

type Point interface {
	Y() float64
	X() int64
}

type Series interface {
	Title() string

	// Id should be unique in a given SeriesSet, and should be consistent
	// across refreshes to ensure things like coloring and ordering stay
	// consistent.  It must not be NoSeries.
	Id() SeriesId

	// Details() []string // TODO: show for more details on hover on the line?

	// Points *must* have a domain that is monotinically increasing
	Points() []Point
}

// RangeScale maps a platonic range to another platonic range.
// Use it to do stuff like apply log scales
type RangeScale func(float64) float64

type PlatonicAxes struct {
	DomainMin, DomainMax int64
	RangeMin, RangeMax float64
}
func AutoAxes() PlatonicAxes {
	return PlatonicAxes{
		// NB(sollyross): Min gets set to Max, and vice versa, so that
		// anything is automatically less/more (respectively) than them.
		DomainMin: math.MaxInt64,
		DomainMax: math.MinInt64,
		RangeMin: math.Inf(1),
		RangeMax: math.Inf(-1),
	}
}
func (a PlatonicAxes) WithPreviousRange(oldAxes PlatonicAxes) PlatonicAxes {
	// not a pointer so we get a copy
	a.RangeMin = oldAxes.RangeMin
	a.RangeMax = oldAxes.RangeMax
	return a
}

type PlatonicGraph struct {
	PlatonicAxes

	Series SeriesSet
}

func DataToPlatonicGraph(seriesSet SeriesSet, baseAxes PlatonicAxes) *PlatonicGraph {
	res := &PlatonicGraph{
		Series: seriesSet,
		PlatonicAxes: baseAxes,
	}

	for _, series := range res.Series {
		pts := series.Points()
		if len(pts) > 0 {
			minDomain := pts[0].X()
			maxDomain := pts[len(pts)-1].X()

			if minDomain < res.DomainMin {
				res.DomainMin = minDomain
			}
			if maxDomain > res.DomainMax {
				res.DomainMax = maxDomain
			}

			for _, pt := range pts {
				val := pt.Y()
				if val < res.RangeMin {
					res.RangeMin = val
				}
				if val > res.RangeMax {
					res.RangeMax = val
				}
			}
		}
	}

	return res
}

func (g PlatonicGraph) ScalePlatonicToScreen(scale RangeScale, size ScreenSize) (func(int64) Column, func(float64) Row) {
	// since the valid values for rows are [0, Rows), subtract one to make sure
	// that g.DomainMax --> Rows-1 < Rows, and similarly for cols

	// avoid issues with flat lines
	domainDiff := g.DomainMax - g.DomainMin
	rangeDiff := scale(g.RangeMax) - scale(g.RangeMin)
	if domainDiff == 0 {
		domainDiff = 1
	}
	if rangeDiff == 0 {
		rangeDiff = 1
	}
	domainScaleFactor := float64(size.Cols-1)/float64(domainDiff)
	rangeScaleFactor := float64(size.Rows-1)/float64(rangeDiff)

	// TODO: is this rounding necessary for most data?
	domain := func(x int64) Column {
		return Column(math.Round(float64(x-g.DomainMin)*domainScaleFactor))
	}
	rng := func(y float64) Row {
		return Row(math.Round(float64(scale(y-g.RangeMin))*rangeScaleFactor))
	}

	return domain, rng
}

func (g *PlatonicGraph) ToScreen(scale RangeScale, size ScreenSize) *ScreenGraph {
	// first, figure out our scaling functions
	domain, rng := g.ScalePlatonicToScreen(scale, size)


	// then, figure out our points -- map the X and Y for each point, figure
	// out if the X falls into the last X bucket (in which case take the
	// average)

	outSeries := make([]ScreenSeries, len(g.Series))

	for i, inSeries := range g.Series {
		var pts []PixelPoint

		// we might need to average a point, so keep this around to figure that out
		var lastPt *PixelPoint

		for _, inPoint := range inSeries.Points() {
			inX, inY := inPoint.X(), inPoint.Y()

			col, row := domain(inX), rng(inY)
			if lastPt != nil {
				if lastPt.Col == col {
					// if we need to accumulate in the last row, just do that
					lastPt.Row += row
					lastPt.OriginalPoints = append(lastPt.OriginalPoints, inPoint)
					continue
				}

				// if we were accumulating, do the final averaging
				if len(lastPt.OriginalPoints) > 1 {
					lastPt.Row /= Row(len(lastPt.OriginalPoints))  // WHY? WHY CAN'T GO'S TYPE SYSTEM UNDERSTAND THE EXISTENCE OF UNITLESS TYPES?
				}
			}

			// in any case, if we've hit here, this is a new point,
			// so reset our tracking and append the new point
			pts = append(pts, PixelPoint{Row: row, Col: col, OriginalPoints: []Point{inPoint}})
			lastPt = &pts[len(pts)-1]
		}

		// handle the very last point
		if len(lastPt.OriginalPoints) > 1 {
			lastPt.Row /= Row(len(lastPt.OriginalPoints))
		}

		outSeries[i] = ScreenSeries{
			Id: inSeries.Id(),
			Points: pts,
		}
	}


	return &ScreenGraph{
		Series: outSeries,
		ScreenSize: size,
	}
}

type Row int
type Column int

// RangeProjector maps a platonic range into a screen range.
// Use it to convert data points into pixels
type RangeProjector func(float64) Row

// DomainProjector maps a platonic domain into a screen domain.
// Use it to convert data points into pixels
type DomainProjector func(int64) Column

type PixelPoint struct {
	Row Row
	Col Column

	OriginalPoints []Point
}

type ScreenSeries struct {
	Points []PixelPoint
	Id SeriesId
}

type ScreenGraph struct {
	ScreenSize

	Series []ScreenSeries
}

type ScreenSize struct {
	Rows Row
	Cols Column
}

type Cell struct {
	// common path doesn't need to allocate a slice
	IsPoint bool
	Series SeriesId

	MoreSeries []SeriesId
}

func (g *RenderedGraph) setCell(row Row, col Column, isPoint bool, series SeriesId) {
	ind := g.SubCellMapper(row, col, g.ScreenSize)

	// and save it
	cell := &g.Cells[ind]
	if cell.Series == NoSeries {
		cell.Series = series
		cell.IsPoint = isPoint // we're only called on points the first run around, so this is fine
		return
	}

	cell.MoreSeries = append(cell.MoreSeries, series)
}

func (s *ScreenGraph) Render(subCellMapper SubCellMapper) *RenderedGraph {
	res := &RenderedGraph{
		ScreenSize: s.ScreenSize,
		Cells: make([]Cell, int(s.Rows)*int(s.Cols)),
		SubCellMapper: subCellMapper,
	}

	// write out points first so that points are always the "first" series
	// listed.
	for _, series := range s.Series {
		for _, pt := range series.Points {
			res.setCell(pt.Row, pt.Col, true, series.Id)
		}
	}

	// write out interpolated lines next
	for _, series := range s.Series {
		if len(series.Points) < 2 {
			// no line drawing if we don't have at least two points ;-)
			continue
		}
		lastPt := series.Points[0]
		for i := 1; i < len(series.Points); i++ {
			pt := series.Points[i]

			rise := float64(pt.Row - lastPt.Row)
			run := float64(pt.Col - lastPt.Col)

			// https://en.wikipedia.org/wiki/Bresenham%27s_line_algorithm,
			// more-or-less
			row, col := lastPt.Row, lastPt.Col
			if math.Abs(run) > math.Abs(rise) {
				slope := rise/run
				slopeErr := 0.0
				var riseInc Row
				if slope > 0 {
					riseInc = 1
				} else {
					riseInc = -1
				}
				for col <= pt.Col {
					res.setCell(row, col, false, series.Id)
					col++
					slopeErr += slope
					if slopeErr >= 0.5 {
						row += riseInc
						slopeErr -= 1.0
					}
				}
			} else {
				slope := run/rise
				slopeErr := 0.0
				var runInc Column
				if slope > 0 {
					runInc = 1
				} else {
					runInc = -1
				}
				for row <= pt.Row {
					res.setCell(row, col, false, series.Id)
					row++
					slopeErr += slope
					if slopeErr >= 0.5 {
						col += runInc
						slopeErr -= 1.0
					}
				}
			}

			lastPt = pt
		}
	}

	return res
}

type SubCellMapper func(row Row, col Column, size ScreenSize) int

type RenderedGraph struct {
	Cells []Cell
	ScreenSize
	SubCellMapper SubCellMapper
}

// conviniently, according to the braille patterns docs (e.g.
// https://en.wikipedia.org/wiki/Braille_Patterns), each position in the
// braille cell is mapped to a bit in the byte, like so:
// 0 3
// 1 4
// 2 5
// 6 7
//
// our data is conviniently laid out to facilitate this.

const (
	brailleBlockStart = '\u2800'
)
// brailleMap maps a column-wise layout to the above braille block layout.
var brailleMap = [8]rune{1<<0, 1<<1, 1<<2, 1<<6, 1<<3, 1<<4, 1<<5, 1<<7}

func DrawBraille(graph *RenderedGraph, output func(row Row, col Column, cell rune, id SeriesId)) {
	currRow := Row(-1)
	currCol := Column(0)
	screenCols := int(graph.Cols) / brailleCellWidth
	for chunkStart := 0; chunkStart < len(graph.Cells); chunkStart += brailleCellPositions {
		if (chunkStart/brailleCellPositions) % screenCols == 0 {
			currRow++
			currCol = 0
		} else {
			currCol++
		}
		targetBits := rune(0)
		var targetId SeriesId
		for cellInd := 0; cellInd < brailleCellPositions; cellInd++ {
			cell := graph.Cells[chunkStart+cellInd]
			if cell.Series == NoSeries {
				continue
			}
			targetBits |= brailleMap[cellInd]

			// we can only have one color, just choose the last one set since
			// it's convinient
			targetId = cell.Series
		}

		if targetBits == 0 {
			output(currRow, currCol, ' ', NoSeries)
			continue
		}

		targetRune := targetBits + brailleBlockStart
		output(currRow, currCol, targetRune, targetId)
	}
}

func BrailleCellMapper(row Row, col Column, size ScreenSize) int {
	// since a screen character is 4 high by 2 wide (ratio/via braille
	// characters), cells are layed out in 2x4 chunks.  Chunks are arraged
	// row-wise (one whole row, then the next) in order to facilitate printing
	// characters, but cells in a chunk are arraged column-wise to match with
	// how the braille patterns unicode characters do it.

	// flip the graph during rendering
	row = size.Rows - 1 - row

	// find the chunk (row-wise)
	chunkRow := int(row/brailleCellHeight)
	chunkCol := int(col/brailleCellWidth)
	chunkStart := (chunkRow*(int(size.Cols)/brailleCellWidth)+chunkCol)*brailleCellPositions

	// find the position in the chunk (column-wise)
	intraChunkRow := int(row%brailleCellHeight)
	intraChunkCol := int(col%brailleCellWidth)
	intraChunkPos := intraChunkCol*brailleCellHeight+intraChunkRow

	// compute the index
	return chunkStart+intraChunkPos
}

func BrailleCellScreenSize(termSize ScreenSize) ScreenSize {
	termSize.Rows *= brailleCellHeight
	termSize.Cols *= brailleCellWidth

	return termSize
}
