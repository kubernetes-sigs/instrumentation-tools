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
	"github.com/gdamore/tcell"
)

// styledSpan describes a span of styled text.
type styledSpan struct {
	val string
	sty tcell.Style
}

// TextBox is a semi-static (i.e. not a text input widget) text container to
// which styled text can be written.  It will automatically wrap text as
// necessary.  If the text doesn't fit, some will be scrolled out of view.
type TextBox struct {
	wrapper textWrapper
	contents []styledSpan

	pos PositionBox
}

func (t *TextBox) SetBox(box PositionBox) {
	t.pos = box

	t.wrapper.Resize(box.Cols, box.Rows)
}

// WriteString writes the given text to the text box in the given style,
// wrapping & scrolling if necessary.
func (t *TextBox) WriteString(str string, sty tcell.Style) {
	t.contents = append(t.contents, styledSpan{val: str, sty: sty})
}

func (t *TextBox) FlushTo(screen tcell.Screen) {
	if t.pos.Rows == 0 || t.pos.Cols == 0 {
		// bail, we've effectively been asked not to render
		return
	}
	t.wrapper.CursorGoTo(0, 0)
	for _, chunk := range t.contents {
		t.wrapper.WriteString(chunk.val, chunk.sty)
	}
	t.wrapper.EraseDown()
	t.wrapper.FlushTo(screen, t.pos.StartCol, t.pos.StartRow)
}
