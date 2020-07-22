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

type Labeling struct {
	DomainLabeler
	RangeLabeler
	LineSize int
}
type marginInfo struct {
	domLbls, rngLbls []string
	marginRows       Row
	marginCols       Column
}
func (l Labeling) labels(domTicks []int64, rngTicks []float64, lineSize int) marginInfo {
	// NB(sollyross): there's a bit of weirdness going on in that non-displayed
	// labels can affect the width, but such is life when we collapse multiple
	// ticks into a single cell.
	res := marginInfo{
		domLbls: make([]string, len(domTicks)),
		rngLbls: make([]string, len(rngTicks)),
	}
	for i, tick := range domTicks {
		lbl := l.DomainLabeler(tick)
		res.domLbls[i] = lbl
		if len(lbl) > int(res.marginRows) {
			res.marginRows = Row(len(lbl))
		}
	}
	for i, tick := range rngTicks {
		lbl := l.RangeLabeler(tick)
		res.rngLbls[i] = lbl
		if len(lbl) > int(res.marginCols) {
			res.marginCols = Column(len(lbl))
		}
	}
	res.marginRows += Row(lineSize)
	res.marginCols += Column(lineSize)
	return res
}

type TickScaling struct {
	RangeScale RangeScale

	RangeDensity int
	DomainDensity int
}

func EvenlySpacedTicks(graph *PlatonicGraph, outerSize ScreenSize, scale TickScaling, labels Labeling) *ScreenTicks {
	// first, figure out the ticks in their platonic positions


	// use the data to inspire the number of domain ticks, putting actual data points into buckets
	var platDomTicks []int64
	{
		domTicksBase := int64(int(outerSize.Cols) / scale.DomainDensity)
		if domTicksBase == 0 {
			domTicksBase = 1
		}

		numTicks := int64(math.MinInt64)
		for _, series := range graph.Series {
			numPts := len(series.Points())
			if int64(numPts) > numTicks {
				numTicks = int64(numPts)
			}
		}
		if numTicks > domTicksBase {
			numTicks = domTicksBase
		}
		if numTicks == 0 || numTicks == math.MinInt64 {
			numTicks = 1
		}

		platDomInc := (graph.DomainMax - graph.DomainMin)/domTicksBase
		if platDomInc == 0 {
			platDomInc = graph.DomainMax // just two ticks
		}
		platDomTicks = make([]int64, numTicks)
		for i := range platDomTicks {
			platDomTicks[i] = graph.DomainMin+(platDomInc*int64(i))
		}
		// always put a tick @ max
		if len(platDomTicks) > 0 && platDomTicks[len(platDomTicks)-1] != graph.DomainMax {
			platDomTicks = append(platDomTicks, graph.DomainMax)
		}
	}

	// evenly space the range ticks, always including min and max
	var platRngTicks []float64 // TODO: size this appropriately?
	{
		rngTicksBase := float64(int(outerSize.Cols) / scale.RangeDensity)
		if rngTicksBase == 0 {
			rngTicksBase = 1
		}


		platRngInc := (graph.RangeMax - graph.RangeMin)/rngTicksBase
		if platRngInc == 0 {
			platRngInc = graph.RangeMax // just two ticks
		}
		for x := graph.RangeMin; x <= graph.RangeMax; x += platRngInc {
			platRngTicks = append(platRngTicks, x)
		}
		// always put a tick @ max
		if len(platRngTicks) > 0 && platRngTicks[len(platRngTicks)-1] != graph.RangeMax {
			platRngTicks = append(platRngTicks, graph.RangeMax)
		}
	}

	// then, compute the labels so that we can figure out the margins required
	// when computing the screen size
	labelInfo := labels.labels(platDomTicks, platRngTicks, labels.LineSize)

	// then, map to screen positions
	innerSize := outerSize
	innerSize.Rows -= labelInfo.marginRows
	innerSize.Cols -= labelInfo.marginCols

	// fix to zero so that we can bail in other code
	if innerSize.Rows < 0 || innerSize.Cols < 0 {
		innerSize.Rows = 0
		innerSize.Cols = 0
	}

	domain, rng := graph.ScalePlatonicToScreen(scale.RangeScale, innerSize)

	ticks := &ScreenTicks{
		InnerGraphSize: innerSize,
		MarginRows: labelInfo.marginRows,
		MarginCols: labelInfo.marginCols,
		LineSize: labels.LineSize,
	}

	{
		var lastTick *DomainTick
		for i, platTick := range platDomTicks {
			col := domain(platTick)
			// discard duplicate ticks
			if lastTick != nil && lastTick.Col == col {
				continue
			}
			ticks.DomainTicks = append(ticks.DomainTicks, DomainTick{
				Col: col, Value: platTick, Label: labelInfo.domLbls[i],
			})
			lastTick = &ticks.DomainTicks[len(ticks.DomainTicks)-1]
		}
	}
	{
		var lastTick *RangeTick
		for i, platTick := range platRngTicks {
			row := rng(platTick)
			// discard duplicate ticks
			if lastTick != nil && lastTick.Row == row {
				continue
			}
			ticks.RangeTicks = append(ticks.RangeTicks, RangeTick{
				Row: innerSize.Rows - row - 1, // invert the row
				Value: platTick, Label: labelInfo.rngLbls[i],
			})
			lastTick = &ticks.RangeTicks[len(ticks.RangeTicks)-1]
		}
	}

	return ticks
}

type DomainTick struct {
	Col Column
	Value int64
	Label string
}

type RangeTick struct {
	Row Row
	Value float64
	Label string
}

type ScreenTicks struct {
	DomainTicks []DomainTick
	RangeTicks []RangeTick

	InnerGraphSize ScreenSize
	MarginRows Row
	MarginCols Column
	LineSize int
}

type DomainLabeler func(int64) string
type RangeLabeler func(float64) string

type LabelInfo struct {
	DomainLabels, RangeLabels []string

	MarginRows Row
	MarginCols Column
}

type AxisCellKind int
const (
	DomainTickKind AxisCellKind = iota
	RangeTickKind
	YAxisKind
	XAxisKind
	AxisCornerKind
	LabelKind
)

func DrawAxes(ticks *ScreenTicks, output func(row Row, col Column, cell rune, kind AxisCellKind)) {
	// first, draw axis lines
	{
		col := ticks.MarginCols - 1
		for row := Row(0); row < ticks.InnerGraphSize.Rows; row++ {
			output(row, col, ' ', YAxisKind)
		}
	}
	{
		row := ticks.InnerGraphSize.Rows
		for graphCol := Column(0); graphCol < ticks.InnerGraphSize.Cols; graphCol++ {
			// start at the first of the "graph" columns (i.e. the first non-margin column) --
			// technically, we also want to cover the last margin column (the axis), but that's
			// the corner piece, handled below).
			col := graphCol + ticks.MarginCols
			output(row, col, ' ', XAxisKind)
		}
	}

	// then, draw ticks & labels
	{
		col := ticks.MarginCols - 1  // start at the line position, let users offset if they want
		for _, tick := range ticks.RangeTicks {
			// tick
			output(tick.Row, col, ' ', RangeTickKind)

			// label, right-justified
			lblPos := col-Column(len(tick.Label))
			// TODO: combining chars?
			for _, rn := range tick.Label {
				output(tick.Row, lblPos, rn, LabelKind)
				lblPos++
			}
		}
	}
	{
		row := ticks.InnerGraphSize.Rows // start at the line position, let users offset if they want
		for _, tick := range ticks.DomainTicks {
			// tick
			output(row, tick.Col+ticks.MarginCols-1, ' ', DomainTickKind)

			// label, top-justified
			lblPos := row+Row(ticks.LineSize)
			// TODO: combining chars?
			for _, rn := range tick.Label {
				output(lblPos, tick.Col+ticks.MarginCols-1, rn, LabelKind)
				lblPos++
			}
		}
	}

	// finally the 0, 0 corner (after ticks so if we do something special to
	// overwrite the axis lines for ticks, the corner character has the final
	// say)
	output(ticks.InnerGraphSize.Rows, ticks.MarginCols-1, ' ', AxisCornerKind)
}
