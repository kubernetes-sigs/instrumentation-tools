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
	"sync"
	"syscall"
	"fmt"
	"unicode/utf8"

	"github.com/gdamore/tcell"
	"github.com/c-bata/go-prompt"
)

type screenIsh interface {
	ShowCursor(int, int)
	HideCursor()
	RequestRepaint()
}

type cellWriter struct {
	screen screenIsh
	startRow, startCol int

	textWrapper

	currentStyle tcell.Style
}

func (w *cellWriter) SetBox(box PositionBox) {
	w.startCol = box.StartCol
	w.startRow = box.StartRow
	w.rows = box.Rows
	w.cols = box.Cols

	w.buf.Resize(box.Cols, box.Rows)
}

func (w *cellWriter) WriteRaw(data []byte) {
	// only ever used to write '\n', only handle that case
	if len(data) == 1 && data[0] == '\n' {
		w.Newline()
		return
	}
	panic(fmt.Sprintf("non-newline raw write not implemented: %v", data))
}
func (w *cellWriter) Write(data []byte) {
	panic("not used")
}
func (w *cellWriter) WriteRawStr(data string) {
	panic("not used")
}

func (w *cellWriter) WriteStr(data string) {
	w.WriteString(data, w.currentStyle)
}
func (w *cellWriter) Flush() error {
	w.screen.RequestRepaint()
	return nil
}
func (w *cellWriter) EraseScreen() {
	w.Erase()
}
func (w *cellWriter) ShowCursor() {
	cursorCol, cursorRow := w.CursorPosition()
	w.screen.ShowCursor(w.startCol+cursorCol, w.startRow+cursorRow)
}
func (w *cellWriter) HideCursor() {
	w.screen.HideCursor()
}
func (w *cellWriter) AskForCPR() {
	panic("not used")
}
func (w *cellWriter) SaveCursor() {
	panic("not used")
}
func (w *cellWriter) UnSaveCursor() {
	panic("not used")
}

func (w *cellWriter) SetTitle(title string) {
	// no-op
}
func (w *cellWriter) ClearTitle() {
	// no-op
}
func (w *cellWriter) SetColor(fg, bg prompt.Color, bold bool) {
	// the normal colors cast almost directly to tcell colors
	// ("default" is iota in prompt, but black is iota in tcell)
	w.currentStyle = tcell.StyleDefault.Bold(bold)
	if fg != prompt.DefaultColor {
		w.currentStyle = w.currentStyle.Foreground(tcell.Color(fg-1))
	}
	if bg != prompt.DefaultColor {
		w.currentStyle = w.currentStyle.Background(tcell.Color(bg-1))
	}
}


type screenParser struct {
	size *prompt.WinSize
	// NB(directxman12): go-prompt assumes shortcut keys and things like enter come in on their
	// own event, so we send keys individually and collapse non-newline runes in read inline
	evts chan *tcell.EventKey
	leftOvers []byte
	mu sync.Mutex
}
func (screenParser) Setup() error {
	return nil
}
func (screenParser) TearDown() error {
	return nil
}
func (p *screenParser) GetWinSize() *prompt.WinSize {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.size
}
func (p *screenParser) Resize(size *prompt.WinSize) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.size = size
}

func (p *screenParser) Read() ([]byte, error) {
	// check if we had accumulated some normal bytes and then hit a newline or
	// special key
	if p.leftOvers != nil {
		res := p.leftOvers
		p.leftOvers = nil
		return res, nil
	}

	var res []byte
CollapseLoop:
	for {
		// the code from go-prompt normally sets NOBLOCK, so emulate that
		select {
		case evt := <-p.evts:
			// special keys -- these need to be send differently,
			// otherwise go-prompt won't catch them as shortcuts/special actions
			// (e.g. \r --> submit, tab --> complete).  This is normally
			// not a problem, but can happen theoretically if typing too fast
			// or if we synthetically batch up input
			if evt.Key() != tcell.KeyRune {
				bytes := keyToBytes(evt)
				if bytes != nil {
					p.leftOvers = bytes
				}
				break CollapseLoop
			}

			// otherwise, "normal" runes, collapse to smooth things if we get a
			// bunch of input at once
			rn := evt.Rune()
			if rn < utf8.RuneSelf {
				res = append(res, byte(rn))
				continue
			}

			var buf [utf8.UTFMax]byte
			n := utf8.EncodeRune(buf[:], rn)
			res = append(res, buf[:n]...)
		default:
			break CollapseLoop
		}
	}
	// if we had a single special key, treat that as the result
	if len(res) == 0 && len(p.leftOvers) > 0 {
		res = p.leftOvers
		p.leftOvers = nil
	}

	if len(res) > 0 {
		return res, nil
	}
	return nil, syscall.EWOULDBLOCK
}

func (p *screenParser) AddKey(evt *tcell.EventKey) {
	p.evts <- evt
}

func (p *screenParser) AddString(str string) {
	for _, rn := range str {
		p.evts <- tcell.NewEventKey(tcell.KeyRune, rn, 0)
	}
}

func keyToBytes(evt *tcell.EventKey) []byte {
	if evt.Key() == tcell.KeyRune {
		// use it straight-up
		rn := evt.Rune()
		if rn < utf8.RuneSelf {
			return []byte{byte(rn)}
		}
		var buf [utf8.UTFMax]byte
		runeSize := utf8.EncodeRune(buf[:], rn)
		return buf[:runeSize]
	}


	// otherwise, translate tcell special events back to the corresponding go-prompt keys
	key := prompt.NotDefined

	// TODO: this is a bit lazy -- if tcell or go-prompt re-arranges constants, this'll break
	rawKey := evt.Key()

	// easy ranges
	switch {
	case rawKey >= tcell.KeyCtrlA && rawKey <= tcell.KeyCtrlZ:
		key = prompt.Key(rawKey - tcell.KeyCtrlA + 1 /* 0 is prompt.Escape */)
	case rawKey >= tcell.KeyF1 && rawKey <= tcell.KeyF24:
		key = prompt.Key(rawKey - tcell.KeyF1) + prompt.F1
	}

	// direct equivalents without easy ranges
	// NB(sollyross): these go after the ranges because some aliases (tab and escape)
	// are the same key in tcell, but treated differently by go-prompt
	switch rawKey {
	case tcell.KeyTab:
		key = prompt.Tab
	case tcell.KeyCtrlSpace:
		key = prompt.ControlSpace
	case tcell.KeyCtrlBackslash:
		key = prompt.ControlBackslash
	case tcell.KeyCtrlRightSq:
		key = prompt.ControlSquareClose
	case tcell.KeyESC:
		key = prompt.Escape
	case tcell.KeyCtrlCarat:
		key = prompt.ControlCircumflex
	case tcell.KeyCtrlUnderscore:
		key = prompt.ControlUnderscore
	case tcell.KeyHome:
		key = prompt.Home
	case tcell.KeyEnd:
		key = prompt.End
	case tcell.KeyPgUp:
		key = prompt.PageUp
	case tcell.KeyPgDn:
		key = prompt.PageDown
	case tcell.KeyBacktab:
		key = prompt.BackTab
	case tcell.KeyInsert:
		key = prompt.Insert
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		key = prompt.Backspace
	}

	modKey := evt.Modifiers()
	isCtrl := modKey&tcell.ModCtrl != 0
	isShift := modKey&tcell.ModShift != 0

	// ones where different modifiers affect keys
	switch rawKey {
	case tcell.KeyLeft:
		key = prompt.Left
		switch {
		case isCtrl:
			key = prompt.ControlLeft
		case isShift:
			key = prompt.ShiftLeft
		}
	case tcell.KeyRight:
		key = prompt.Right
		switch {
		case isCtrl:
			key = prompt.ControlRight
		case isShift:
			key = prompt.ShiftRight
		}
	case tcell.KeyUp:
		key = prompt.Up
		switch {
		case isCtrl:
			key = prompt.ControlUp
		case isShift:
			key = prompt.ShiftUp
		}
	case tcell.KeyDown:
		key = prompt.Down
		switch {
		case isCtrl:
			key = prompt.ControlDown
		case isShift:
			key = prompt.ShiftDown
		}
	case tcell.KeyDelete:
		key = prompt.Delete
		switch {
		case isCtrl:
			key = prompt.ControlDelete
		case isShift:
			key = prompt.ShiftDelete
		}
	}
	// this is hillariously inefficient since go-prompt immediately reverses
	// it, but it's what we're stuck with
	for _, seq := range prompt.ASCIISequences {
		if seq.Key == key {
			return seq.ASCIICode
		}
	}
	return nil
}
