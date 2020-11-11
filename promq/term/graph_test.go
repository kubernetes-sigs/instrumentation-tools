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

package term_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/instrumentation-tools/promq/term"
	"sigs.k8s.io/instrumentation-tools/promq/term/plot"
)

type trivialPoint struct {
	x int64
	y float64
}
func (p trivialPoint) X() int64 {
	return p.x
}
func (p trivialPoint) Y() float64 {
	return p.y
}

type trivialSeries struct {
	title string
	id plot.SeriesId
	pts []plot.Point
}
func (s trivialSeries) Title() string {
	return s.title
}
func (s trivialSeries) Id() plot.SeriesId {
	return s.id
}
func (s trivialSeries) Points() []plot.Point {
	return s.pts
}

var samplePlatonicGraph = plot.DataToPlatonicGraph(
	plot.SeriesSet{trivialSeries{
		title: "linear 1",
		id: plot.SeriesId(1),
		pts: []plot.Point{
			trivialPoint{0, 0},
			trivialPoint{2, 3.0},
			trivialPoint{3, 8.7},
			trivialPoint{17, 0.666666666667},
		},
	}},
	plot.AutoAxes(),
)
func trivialDomLabeler(x int64) string {
	return fmt.Sprintf("%2d", x)
}
func trivialRngLabeler(y float64) string {
	return fmt.Sprintf("%2.2g", y)
}

var _ = Describe("The Graph widget", func() {
	Context("when dealing with size & content corner cases", func() {
		It("should skip rendering if given zero columns", func() {
			gr := &term.GraphView{
				Graph: samplePlatonicGraph,
				DomainLabeler: trivialDomLabeler, 
				RangeLabeler: trivialRngLabeler,
			}

			gr.SetBox(term.PositionBox{Rows: 1, Cols: 0})
			Expect(gr).To(DisplayLike(1, 10, ""))
		})

		It("should skip rendering if given zero rows", func() {
			gr := &term.GraphView{
				Graph: samplePlatonicGraph,
				DomainLabeler: trivialDomLabeler,
				RangeLabeler: trivialRngLabeler,
			}

			gr.SetBox(term.PositionBox{Rows: 0, Cols: 100})
			Expect(gr).To(DisplayLike(1, 10, ""))
		})

		It("should skip drawing without any graph", func() {
			gr := &term.GraphView{Graph: nil}
			gr.SetBox(term.PositionBox{Rows: 10, Cols: 10})
			Expect(gr).To(DisplayLike(10, 10, ""))
		})
	})

	It("should start writing at the position specified by its box", func() {
		gr := &term.GraphView{
			Graph: samplePlatonicGraph,
			DomainLabeler: trivialDomLabeler,
			RangeLabeler: trivialRngLabeler,
			DomainTickSpacing: 4,
			RangeTickSpacing: 3,
		}

		gr.SetBox(term.PositionBox{
			StartRow: 1, StartCol: 2,
			Rows: 10, Cols: 12,
		})

		Expect(gr).To(DisplayLike(15, 13, 
			"               "+
			"  8.7┨ ⢸       "+
			"  6.5┨ ⢸       "+
			"     ┃ ⡜       "+
			"  4.3┨ ⡇       "+
			"  2.2┨ ⡇       "+
			"     ┃⢰⠁       "+
			"    0┨⡎      ⠐ "+
			"     ┗━┯━┯━━┯━ "+
			"         1  1  "+
			"     0 5 0  7  "+
			"               "+
			"               "))
	})

	Context("when rendering axes", func() {
		It("should use the provided tick labelers to label the axes", func() {
			gr := &term.GraphView{
				Graph: samplePlatonicGraph,
				DomainLabeler: func(x int64) string { return "X" },
				RangeLabeler: func(y float64) string { return "Y" },
			}

			gr.SetBox(term.PositionBox{
				Rows: 4, Cols: 10,
			})
			Expect(gr).To(DisplayLike(10, 4,
				" ┃ ⡸⠉⠉⠉⠉⠉⠉"+
				"Y┨⡠⠃     ⠠"+
				" ┗━━━━━━┯━"+
				" X      X "))

		})
		It("should use the provided tick spacing to space X & Y axis ticks", func() {
			gr := &term.GraphView{
				Graph: samplePlatonicGraph,

				// 1-char labelers to just test the tick spacing
				DomainLabeler: func(x int64) string { return "X" },
				RangeLabeler: func(y float64) string { return "Y" },
				DomainTickSpacing: 3,
				RangeTickSpacing: 4,
			}

			gr.SetBox(term.PositionBox{
				Rows: 10, Cols: 10,
			})
			Expect(gr).To(DisplayLike(10, 10,
				"Y┨ ⢸      "+
				" ┃ ⢸      "+
				" ┃ ⡜      "+
				"Y┨ ⡇      "+
				" ┃ ⡇      "+
				" ┃⢀⠇      "+
				" ┃⢸       "+
				"Y┨⡎      ⠐"+
				" ┗━┯━┯━━┯━"+
				" X X X  X "))
		})
	})
})
