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

var _ = Describe("The TextBox widget", func() {
	It("should still save text written before the size is set", func() {
		box := &term.TextBox{}
		box.WriteString("the time has come, the walrus said, to talk of many things", tcell.StyleDefault)
		box.SetBox(term.PositionBox{
			Rows: 1, Cols: 200,
		})
		Expect(box).To(DisplayLike(200, 1, "the time has come, the walrus said, to talk of many things"))
	})

	It("should handle text with newlines properly, converting them to proper cursor movement", func() {
		box := &term.TextBox{}
		box.WriteString(
`of shoes, and ships, and sealing wax,
    of cabbages, and kings,
and why the sea is boiling hot
    and whether pigs have wings.`, tcell.StyleDefault)

		box.SetBox(term.PositionBox{
			Rows: 4, Cols: 38,
		})

		Expect(box).To(DisplayLike(38, 4,
			// NB: whitespace is significant here
			"of shoes, and ships, and sealing wax, "+
			"    of cabbages, and kings,           "+
			"and why the sea is boiling hot        "+
			"    and whether pigs have wings.      "))
		
	})

	PIt("should handle tabs properly, accounting for cursor movement properly", func() {

	})

	It("should support writing different text spans with different styles", func() {
		box := &term.TextBox{}
		box.WriteString("but ", tcell.StyleDefault.Foreground(tcell.ColorBlue))
		box.WriteString("wait", tcell.StyleDefault.Foreground(tcell.ColorRed))

		box.SetBox(term.PositionBox{
			Rows: 1, Cols: 8,
		})

		Expect(box).To(DisplayWithStyle(8, 1,
			"but ", tcell.StyleDefault.Foreground(tcell.ColorBlue),
			"wait ", tcell.StyleDefault.Foreground(tcell.ColorRed),
		))
	})

	It("should wrap lines that are longer than the width", func() {
		box := &term.TextBox{}
		box.WriteString("a bit, the oysters cried, before we have our chat", tcell.StyleDefault)

		By("using a box that's smaller than the screen, so that we can see the wrapping in the output")
		box.SetBox(term.PositionBox{Rows: 2, Cols: 25})
		Expect(box).To(DisplayLike(30, 2, "a bit, the oysters cried,      before we have our chat"))
	})

	It("should scroll if the contents are too large for the box", func() {
		box := &term.TextBox{}
		box.WriteString("a loaf of bread, the walrus said, is what we chiefly need: pepper and vinegar besides, Are very good indeed, now if you're ready, oysters dear, we can begin to feed.", tcell.StyleDefault)

		By("using a box that's smaller than the screen, so that we can see the wrapping in the output")
		box.SetBox(term.PositionBox{Rows: 2, Cols: 29})
		Expect(box).To(DisplayLike(32, 3, "you're ready, oysters dear, w   e can begin to feed."))
	})

	It("should properly wrap even if the box changes size", func() {
		box := &term.TextBox{}
		box.WriteString("but not on us!", tcell.StyleDefault)

		By("setting a initial size with no wrapping")
		box.SetBox(term.PositionBox{Rows: 1, Cols: 14})
		Expect(box).To(DisplayLike(14, 1, "but not on us!"))

		By("changing to a size with line wrapping")
		box.SetBox(term.PositionBox{Rows: 4, Cols: 4})
		Expect(box).To(DisplayLike(6, 4, "but   not   on u  s!"))
	})
	
	It("should skip rendering if given zero columns", func() {
		box := &term.TextBox{}
		box.WriteString("the oysters cried", tcell.StyleDefault)

		box.SetBox(term.PositionBox{Rows: 1, Cols: 0})
		Expect(box).To(DisplayLike(1, 10, ""))
	})

	It("should skip rendering if given zero rows", func() {
		box := &term.TextBox{}
		box.WriteString("turning a little blue.", tcell.StyleDefault)

		box.SetBox(term.PositionBox{Rows: 0, Cols: 100})
		Expect(box).To(DisplayLike(1, 10, ""))
	})

	It("should start writing at the position specified by its box", func() {
		box := &term.TextBox{}
		box.WriteString("after", tcell.StyleDefault)

		box.SetBox(term.PositionBox{
			StartRow: 5, StartCol: 5,
			Rows: 1, Cols: 10,
		})

		Expect(box).To(DisplayLike(20, 6,
			"                    "+
			"                    "+
			"                    "+
			"                    "+
			"                    "+
			"     after          "))
	})

	PIt("should handle multi-byte-single-rune contents", func() {

	})

	PIt("should handle combining characters and such", func() {

	})
})
