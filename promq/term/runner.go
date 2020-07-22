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
	"context"

	"github.com/gdamore/tcell"
)

type View interface {
	Flushable
	Resizable
}

type Runner struct {
	screen tcell.Screen
	screenMu sync.Mutex

	KeyHandler func(*tcell.EventKey)
}

func (r *Runner) Run(ctx context.Context, initialView View) error {
	screen, err := tcell.NewScreen()
	if err != nil {
		return err
	}
	screen.Init()
	defer screen.Fini()

	r.screenMu.Lock()
	r.screen = screen
	r.screenMu.Unlock()

	screenCols, screenRows := screen.Size()
	mainView := initialView
	go func() {
		for evt := screen.PollEvent(); evt != nil; evt = screen.PollEvent() {
			switch evt := evt.(type) {
			case *tcell.EventKey:
				r.KeyHandler(evt)
				continue
			case *tcell.EventInterrupt:
				newView, hasNewView := evt.Data().(View)
				if hasNewView {
					// clearing is less efficient, but means we
					// don't get weird artifacts from the sidebar resizing, etc
					screen.Clear()
					mainView = newView
					mainView.SetBox(PositionBox{Cols: screenCols, Rows: screenRows})
				}
				// continue below
			case *tcell.EventResize:
				screenCols, screenRows = evt.Size()
				if mainView != nil {
					mainView.SetBox(PositionBox{Cols: screenCols, Rows: screenRows})
				}
				screen.Clear()
				// continue below
			default:
				return
			}

			if mainView == nil {
				continue
			}
			mainView.FlushTo(screen)
			screen.Show()
		}
	}()

	<-ctx.Done()

	return nil
}

func (r *Runner) RequestRepaint() {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.PostEvent(tcell.NewEventInterrupt(nil))
}

func (r *Runner) RequestUpdate(newView View) {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.PostEvent(tcell.NewEventInterrupt(newView))
}

func (r *Runner) ShowCursor(col, row int) {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.ShowCursor(col, row)
}
func (r *Runner) HideCursor() {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.HideCursor()
}
