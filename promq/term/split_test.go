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
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/gdamore/tcell"

    "sigs.k8s.io/instrumentation-tools/promq/term"
)

type flushableTestView struct {
	term.StaticResizable
	FlushedTo tcell.Screen
}

func (v *flushableTestView) FlushTo(screen tcell.Screen) {
	v.FlushedTo = screen
}

var _ = Describe("StaticResizable", func() {
	It("should record the size it was sent", func() {
		resizable := &term.StaticResizable{}
		targetBox := term.PositionBox{
			StartRow: 1,
			StartCol: 2,
			Rows: 3,
			Cols: 4,
		}
		resizable.SetBox(targetBox)
		Expect(resizable.PositionBox).To(Equal(targetBox), "recorded box should equal the passed in one")
	})
})

var _ = Describe("SplitView", func() {
	var (
		view term.SplitView
		dockedView term.StaticResizable
		flexedView term.StaticResizable
	)
	BeforeEach(func() {
		dockedView = term.StaticResizable{}
		flexedView = term.StaticResizable{}
		view = term.SplitView{
			Docked: &dockedView,
			Flexed: &flexedView,
		}
	})

	Context("when positioning the docked content", func() {
		BeforeEach(func() {
			view.DockSize = 10
		})

		It("should support placing a full-width pane on the bottom", func() {
			view.Dock = term.PosBelow
			view.SetBox(term.PositionBox{
				StartRow: 10, StartCol: 20,
				Rows: 100, Cols: 200,
			})

			Expect(dockedView.PositionBox).To(Equal(term.PositionBox{
				// full width
				StartCol: 20, Cols: 200,

				// bottom 10 cols
				StartRow: 100, Rows: 10,
			}))
		})

		It("should support placing a full-width pane at the top", func() {
			view.Dock = term.PosAbove
			view.SetBox(term.PositionBox{
				StartRow: 10, StartCol: 20,
				Rows: 100, Cols: 200,
			})

			Expect(dockedView.PositionBox).To(Equal(term.PositionBox{
				// full width
				StartCol: 20, Cols: 200,

				// top 10 cols
				StartRow: 10, Rows: 10,
			}))
		})

		It("should support placing a full-height pane on the left", func() {
			view.Dock = term.PosLeft
			view.SetBox(term.PositionBox{
				StartRow: 10, StartCol: 20,
				Rows: 100, Cols: 200,
			})

			Expect(dockedView.PositionBox).To(Equal(term.PositionBox{
				// full height
				StartRow: 10, Rows: 100,

				// left 10 cols
				StartCol: 20, Cols: 10,
			}))
		})

		It("should support placing a full-height pane on the right", func() {
			view.Dock = term.PosRight
			view.SetBox(term.PositionBox{
				StartRow: 10, StartCol: 20,
				Rows: 100, Cols: 200,
			})

			Expect(dockedView.PositionBox).To(Equal(term.PositionBox{
				// full height
				StartRow: 10, Rows: 100,

				// right 10 cols
				StartCol: 210, Cols: 10,
			}))
		})
	})

	Context("when docking on the top or bottom", func() {
		BeforeEach(func() {
			view.Dock = term.PosAbove
		})

		Context("the docked pane", func() {
			It("should never be larger than the containing rows, leaving at least 1 row for the flexed view", func() {
				view.DockSize = 100
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 50, Cols: 50,
				})

				Expect(dockedView.Rows).To(Equal(49))
			})
			It("should never have fewer than 0 rows", func() {
				view.DockSize = 100
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 0, Cols: 50,
				})

				Expect(dockedView.Rows).To(Equal(0))
			})
			It("should be capped by the max dock percent on small screens", func() {
				view.DockSize = 40
				view.DockMaxPercent = 50
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 50, Cols: 50,
				})
				Expect(dockedView.Rows).To(Equal(25))
			})
		})
	})

	Context("when docking on the left or right", func() {
		BeforeEach(func() {
			view.Dock = term.PosLeft
		})

		Context("the docked pane", func() {
			It("should never be larger than the containing cols, leaving at least 1 col for the flexed view", func() {
				view.DockSize = 100
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 50, Cols: 50,
				})

				Expect(dockedView.Cols).To(Equal(49))
			})
			It("should never have fewer than 0 cols", func() {
				view.DockSize = 100
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 50, Cols: 0,
				})

				Expect(dockedView.Cols).To(Equal(0))
			})
			It("should be capped by the max dock percent on small screens", func() {
				view.DockSize = 40
				view.DockMaxPercent = 50
				view.SetBox(term.PositionBox{
					StartRow: 0, StartCol: 0,
					Rows: 50, Cols: 50,
				})
				Expect(dockedView.Cols).To(Equal(25))
			})
		})
	})

	It("should flush both parts of the split, if flushable, when asked to flush", func() {
		dockedView := &flushableTestView{}
		flexedView := &flushableTestView{}
		view = term.SplitView{
			Docked: dockedView,
			Flexed: flexedView,
		}

		screen := tcell.NewSimulationScreen("")
		view.FlushTo(screen)

		Expect(dockedView.FlushedTo).To(BeIdenticalTo(screen))
		Expect(flexedView.FlushedTo).To(BeIdenticalTo(screen))
	})
})
