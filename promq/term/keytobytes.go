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
	"github.com/c-bata/go-prompt"
)

func nonRuneKeyToBytes(evt *tcell.EventKey) []byte {
	if evt.Key() == tcell.KeyRune {
		panic("just use the rune 🤦")
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

