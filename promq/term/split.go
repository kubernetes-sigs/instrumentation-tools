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

// Flushable contains content that can be flushed to a screen.
type Flushable interface {
	// FlushTo flushes content to the screen.  It should only write to the
	// areas of the screen that it has been assigned to (generally via being
	// Resizable).
	FlushTo(screen tcell.Screen)
}

// DockPos indicates which side of the split-region the "Docked"/fixed-size section
// of a split is anchored to.
type DockPos int
const (
	// PosBelow anchors to the bottom
	PosBelow DockPos = iota
	// PosAbove anchors to the top
	PosAbove
	// PosLeft anchors to the left
	PosLeft
	// PosRight anchors to the right
	PosRight
)

// Resizable widgets know how to receive a section of the screen that they're
// supposed to write to, and resize their content to fit that section.
type Resizable interface {
	// SetBox sets the size that this widget should fill.  This is *not* an
	// indication that the content should be drawn to the screen (that's what
	// Flushable is for).
	SetBox(PositionBox)
}

// PositionBox describes a region of the screen.
type PositionBox struct {
	// StartCol and StartRow indicate the starting row and column
	// (zero-indexed) of this region.
	StartCol, StartRow int
	// Cols and Rows indicate the count of columns in this region,
	// and will be non-zero positive numbers.
	Cols, Rows int
}

// SplitView is a "view" that, given an overall size, knows how to divide that
// size between a "docked" fixed-size pane and another chunk of content.  It's
// used for sidebars, terminals at the bottom of the screen, etc.
type SplitView struct {
	// Dock indicates the position of the fixed-size pane.
	Dock DockPos

	// DockSize indicates the desired size of the fixed-size pane, in rows or
	// columns (depending on where the dock is).
	DockSize int
	// DockMaxPercent caps the actual size of the dock to a percentage of the screen.
	// For instance, if DockSize is 10, the screen size is 20, and DockMaxPercent is 25,
	// the actual dock size used would be 5.
	DockMaxPercent int

	// Docked contains the content for the docked pane.  If also Flushable, it
	// will receive calls to FlushTo as well.
	Docked Resizable
	// Flexed contains the content for the non-docked pane.  If also Flushable,
	// it will receive calls to FlushTo as well.
	Flexed Resizable
}

// capDockSize caps the dock size value to be non-zero/positive, less than the
// screen size, and so that it follows DockMaxPercent, if set.
func (v *SplitView) capDockSize(screenCols, screenRows int) (dockRows, dockCols int) {
	dockRows = v.DockSize
	dockCols = v.DockSize
	if dockRows >= screenRows {
		dockRows = screenRows - 1
	}
	if dockCols >= screenCols {
		dockCols = screenCols - 1
	}
	if dockRows < 0 {
		dockRows = 0
	}
	if dockCols < 0 {
		dockCols = 0
	}

	if v.DockMaxPercent > 0 {
		maxCols := int(float64(screenCols) * float64(v.DockMaxPercent)/100.0)
		if maxCols < dockCols {
			dockCols = maxCols
		}

		maxRows := int(float64(screenRows) * float64(v.DockMaxPercent)/100.0)
		if maxRows < dockRows {
			dockRows = maxRows
		}
	}

	return dockRows, dockCols
}

// dockedBox computes the PositionBox for the docked pane.
func (v *SplitView) dockedBox(screenCols, screenRows int) PositionBox {
	dockRows, dockCols := v.capDockSize(screenCols, screenRows)

	switch v.Dock {
	case PosBelow:
		return PositionBox{StartCol: 0, StartRow: screenRows-dockRows, Cols: screenCols, Rows: dockRows}
	case PosAbove:
		return PositionBox{StartCol: 0, StartRow: 0, Cols: screenCols, Rows: dockRows}
	case PosLeft:
		return PositionBox{StartCol: 0, StartRow: 0, Cols: dockCols, Rows: screenRows}
	case PosRight:
		return PositionBox{StartCol: screenCols-dockCols, StartRow: 0, Cols: dockCols, Rows: screenRows}
	default:
		panic("invalid dock position")
	}
}

// flexedBox computes the PositionBox for the flexed pane.
func (v *SplitView) flexedBox(screenCols, screenRows int) PositionBox {
	dockRows, dockCols := v.capDockSize(screenCols, screenRows)

	switch v.Dock {
	case PosBelow:
		return PositionBox{StartCol: 0, StartRow: 0, Cols: screenCols, Rows: screenRows-dockRows}
	case PosAbove:
		return PositionBox{StartCol: 0, StartRow: dockRows, Cols: screenCols, Rows: screenRows-dockRows}
	case PosLeft:
		return PositionBox{StartCol: dockCols, StartRow: 0, Cols: screenCols-dockCols, Rows: screenRows}
	case PosRight:
		return PositionBox{StartCol: 0, StartRow: 0, Cols: screenCols-dockCols, Rows: screenRows}
	default:
		panic("invalid dock position")
	}
}

func (v *SplitView) SetBox(box PositionBox) {
	docked := v.dockedBox(box.Cols, box.Rows)
	flexed := v.flexedBox(box.Cols, box.Rows)

	docked.StartCol += box.StartCol
	docked.StartRow += box.StartRow
	flexed.StartCol += box.StartCol
	flexed.StartRow += box.StartRow

	v.Docked.SetBox(docked)
	v.Flexed.SetBox(flexed)
}

func (v *SplitView) FlushTo(screen tcell.Screen) {
	if flushable, canFlush := v.Docked.(Flushable); canFlush {
		flushable.FlushTo(screen)
	}
	if flushable, canFlush := v.Flexed.(Flushable); canFlush {
		flushable.FlushTo(screen)
	}
}

// StaticResizable just records the size it was given, without doing anything else.
type StaticResizable struct {
	PositionBox
}

func (r *StaticResizable) SetBox(box PositionBox) {
	r.PositionBox = box
}
