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

package term

import (
    "sigs.k8s.io/instrumentation-tools/promq/term/plot"
	"github.com/gdamore/tcell"
)

type GraphView struct {
	pos PositionBox

	Graph *plot.PlatonicGraph

	DomainLabeler plot.DomainLabeler
	RangeLabeler plot.RangeLabeler
}

func (g *GraphView) SetBox(box PositionBox) {
	g.pos = box
}

func (g *GraphView) FlushTo(screen tcell.Screen) {
	if g.Graph == nil {
		return
	}

	screenSize := plot.ScreenSize{Cols: plot.Column(g.pos.Cols), Rows: plot.Row(g.pos.Rows)}
	scale := func(p float64) float64 { return p }
	axes := plot.EvenlySpacedTicks(g.Graph, screenSize, plot.TickScaling{
		RangeScale: scale,
		DomainDensity: 10, 
		RangeDensity: 10,
	}, plot.Labeling{
		DomainLabeler: g.DomainLabeler,
		RangeLabeler: g.RangeLabeler,
		LineSize: 1,
	})

	if axes.InnerGraphSize.Cols == 0 || axes.InnerGraphSize.Rows == 0 {
		// too small to render, just bail
		return
	}

	plot.DrawAxes(axes, func(row plot.Row, col plot.Column, contents rune, kind plot.AxisCellKind) {
		// TODO: special @ 0, 0
		switch kind {
		case plot.DomainTickKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, '┯', nil, sty)
		case plot.RangeTickKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, '┨', nil, sty)
		case plot.YAxisKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, '┃', nil, sty)
		case plot.XAxisKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, '━', nil, sty)
		case plot.AxisCornerKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, '┗', nil, sty)
		case plot.LabelKind:
			var sty tcell.Style
			screen.SetContent(int(col)+g.pos.StartCol, int(row)+g.pos.StartRow, contents, nil, sty)
		}

	})

	screenGraph := g.Graph.ToScreen(scale, plot.BrailleCellScreenSize(axes.InnerGraphSize))
	renderedGraph := screenGraph.Render(plot.BrailleCellMapper)

	startCol := g.pos.StartCol + int(axes.MarginCols)
	startRow := g.pos.StartRow
	plot.DrawBraille(renderedGraph, func(row plot.Row, col plot.Column, contents rune, id plot.SeriesId) {
		var sty tcell.Style
		if id != plot.NoSeries {
			sty = sty.Foreground(tcell.Color(id % 256))
		}
		screen.SetContent(int(col)+startCol, int(row)+startRow, contents, nil, sty)
	})
}
