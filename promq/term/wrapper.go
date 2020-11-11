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
	"unicode"
	"github.com/mattn/go-runewidth"

	"github.com/gdamore/tcell"
)

// textWrapper wraps written text to a particular view size, and otherwise
// implements cursor movement operations within a text area.  It can be used as
// a base to implement text display widgets, text-input widgets, and terminals.
//
// It contains a zero-indexed cursor.
type textWrapper struct {
	rows, cols int
	buf tcell.CellBuffer
	cursorRow, cursorCol int
}

// Resize sets the size of the widget to the given number of rows and columns.
func (t *textWrapper) Resize(cols, rows int) {
	t.cols = cols
	t.rows = rows
	t.buf.Resize(cols, rows)
}

// Reset clears the contents of the widget and moves the cursor back to (0, 0).
func (t *textWrapper) Reset() {
	t.buf.Fill(' ', tcell.StyleDefault)
	t.cursorRow = 0
	t.cursorCol = 0
}

// Newline clears the reset of the line, and performs both a carriage return
// and line-feed operation.  If the cursor would exceed the height of the wrapper,
// the contents are "scrolled" down by one line, dropping the top line.
func (t *textWrapper) Newline() {
	t.clearLinePart(t.cursorRow, t.cursorCol, t.cols, 1) // clear rest of line
	t.cursorRow++
	t.cursorCol = 0
	if t.cursorRow >= t.rows {
		t.cursorRow = t.rows-1
		t.ScrollDown()
	}
}

// WriteString writes the given string in with the given style, wrapping as
// necessary.  It tries its best to handle combining characters properly.  It
// does not currently remove control sequences from the input string, but that's
// not guaranteed behavior.
func (t *textWrapper) WriteString(str string, sty tcell.Style) {
	// NB(directxman12): technically, this should remove control sequences.
	// Practically, it's not a huge deal
	var charsBuffer []rune
	mainWidth := 0
	for _, rn := range str {
		switch {
		case rn == '\n':
			if len(charsBuffer) > 0 {
				t.writeNextCell(mainWidth, charsBuffer, sty)
				charsBuffer = charsBuffer[:0]
				mainWidth = 0
				// no need to move cursor, newline will do it
			}
			t.Newline()
			continue
		case unicode.IsControl(rn):
			// TODO: do something here?
			continue
		}
		// TODO: zero-width-joiner

		switch width := runewidth.RuneWidth(rn); width {
		case 0: // combinining character
			if len(charsBuffer) == 0 {
				// we don't have a "normal" character already, use space to avoid issues
				charsBuffer = append(charsBuffer, ' ')
				mainWidth = 1
			}
			charsBuffer = append(charsBuffer, rn)
		default: // non-combining character
			if len(charsBuffer) > 0 {
				// we had some deferred main character + combinine characters, print those
				t.writeNextCell(mainWidth, charsBuffer, sty)
				charsBuffer = charsBuffer[:0]
				// TODO: wrap
			}
			charsBuffer = append(charsBuffer, rn)
			mainWidth = width
		}
	}
	if len(charsBuffer) > 0 {
		t.writeNextCell(mainWidth, charsBuffer, sty)
	}
}

// writeNextCell sets the current cell to contain the given base rune &
// *following* combining characters, then advances the cursor by the logical
// "width" of those runes (might be 2+ for certain cases, like full-width CJK).
// If the runes would be too wide to contain the given content, this will wrap
// to the next line.
func (t *textWrapper) writeNextCell(width int, runes []rune, sty tcell.Style) {
	if remaining := t.cols - t.cursorCol; remaining < width {
		// wrap if needed
		t.Newline()
	}
	t.buf.SetContent(t.cursorCol, t.cursorRow, runes[0], runes[1:], sty)
	t.cursorCol += width
}

func (t *textWrapper) FlushTo(screen tcell.Screen, startCol, startRow int) {
	for row := 0; row < t.rows; row++ {
		for col := 0; col < t.cols; col++ {
			if !t.buf.Dirty(col, row) {
				continue
			}
			mainRune, combRunes, style, _ := t.buf.GetContent(col, row)
			screen.SetContent(startCol+col, startRow+row, mainRune, combRunes, style)
		}
	}
}

// clearLinePart clears part of the line of content, starting at start col,
// going until end col in the direction indicated by dir (either 1 or -1).
//
// While not strictly necessary, "dir" makes certain cursor operations easier
// (such as "clear from cursor till beginning of line")
func (t *textWrapper) clearLinePart(row int, startCol, endCol int, dir int) {
	if startCol > endCol && dir > 0 {
		panic("invalid start and end for forward iteration")
	} else if startCol < endCol && dir < 0 {
		panic("invalid start and end for backward iteration")
	} else if dir != 1 && dir != -1 {
		panic("dir must be 1 or -1")
	}
	for c := startCol; c != endCol; c += dir {
		t.buf.SetContent(c, row, ' ', nil, tcell.StyleDefault)
	}
}

// ScrollDown moves the cursor down a line *without* changing the column
// (i.e. linefeed without carriage return).  If we'd scroll off the end of the
// screen, shift all content up by one row instead (an actual "scroll"
// operation).
func (t *textWrapper) ScrollDown() {
	// The operations they want are "index" (this one) and "reverse index" (ScrollUp),
	// which basically move the cursor in the target direction without changing column,
	// doing a scroll operation if we'd go off the screen.

	if t.cursorRow < t.rows-1 {
		t.CursorDown(1)
		return
	}

	// otherwise, move all the lines up one row...
	for r := 1; r < t.rows; r++ {
		for c := 0; c < t.cols; c++ {
			mainRune, combRunes, style, _ := t.buf.GetContent(c, r)
			t.buf.SetContent(c, r-1, mainRune, combRunes, style)
		}
	}
	// ... and clear the last line
	t.clearLinePart(t.rows-1, 0, t.cols, 1)
}

// ScrollUp moves the cursor up a line *without* changing the column (i.e.
// reverse linefeed without carriage return).  If we'd scroll off the top of
// the screen, shift all content down by one row instead (an actual "scroll"
// operation).
func (t *textWrapper) ScrollUp() {
	if t.cursorRow > 0 {
		t.CursorUp(1)
		return
	}

	// otherwise, move all the lines down one row...
	for r := t.rows-2; r <= 0; r-- {
		for c := 0; c < t.cols; c++ {
			mainRune, combRunes, style, _ := t.buf.GetContent(c, r)
			t.buf.SetContent(c, r+1, mainRune, combRunes, style)
		}
	}
	// ... and clear the first line
	t.clearLinePart(0, 0, t.cols, 1)
}

// Erase clears the entire screen.
func (t *textWrapper) Erase() {
	for r := 0; r < t.rows; r++ {
		t.clearLinePart(r, 0, t.cols, 1)
	}
}

// EraseUp clears the screen from the cursor upwards, including everything on
// the current line before the cursor.
func (t *textWrapper) EraseUp() {
	t.clearLinePart(t.cursorRow, t.cursorCol, -1, -1)
	for r := t.cursorRow-1; r >= 0; r-- {
		t.clearLinePart(r, 0, t.cols, 1)
	}
}

// EraseDown clears the screen from the cursor downwards, including everything
// on the current line after the cursor.
func (t *textWrapper) EraseDown() {
	t.clearLinePart(t.cursorRow, t.cursorCol, t.cols, 1)
	for r := t.cursorRow+1; r < t.rows; r++ {
		t.clearLinePart(r, 0, t.cols, 1)
	}
}

// EraseStartOfLine clears from the cursor till the beginning of the line.
func (t *textWrapper) EraseStartOfLine() {
	t.clearLinePart(t.cursorRow, t.cursorCol, -1, -1)
}
// EraseStartOfLine clears from the cursor till the end of the line.
func (t *textWrapper) EraseEndOfLine() {
	t.clearLinePart(t.cursorRow, t.cursorCol, t.cols, 1)
}
// EraseLine clears the current line.
func (t *textWrapper) EraseLine() {
	t.clearLinePart(t.cursorRow, 0, t.cols, 1)
}

// CursorForward moves the cursor forward n columns, stopping at
// the 0th column of the wrapper.
func (t *textWrapper) CursorForward(n int) {
	t.cursorCol += n
	if t.cursorCol >= t.cols {
		t.cursorCol = t.cols-1
	}
}

// CursorForward moves the cursor forward n columns, stopping at
// the last column of the wrapper.
func (t *textWrapper) CursorBackward(n int) {
	t.cursorCol -= n
	if t.cursorCol < 0 {
		t.cursorCol = 0
	}
}

// CursorPosition reports the column and row of the cursor.
func (t *textWrapper) CursorPosition() (col, row int) {
	return t.cursorCol, t.cursorRow
}

// CursorGoTo moves the cursor to the given row and column.
func (t *textWrapper) CursorGoTo(row, col int) {
	t.cursorRow = row
	t.cursorCol = col
}

// CursorDown moves the cursor down n rows, stopping at the bottom of wrapper.
func (t *textWrapper) CursorDown(n int) {
	t.cursorRow += n
	if t.cursorRow >= t.rows {
		t.cursorRow = t.rows-1
	}
}
// CursorUp moves the cursor up n rows, stopping at the top of wrapper.
func (t *textWrapper) CursorUp(n int) {
	t.cursorRow -= n
	if t.cursorRow < 0 {
		t.cursorRow = 0
	}
}
