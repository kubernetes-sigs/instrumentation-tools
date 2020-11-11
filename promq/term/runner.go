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

// View represents a widget -- it can display (Flushable) and be given a size
// (Resizable).
type View interface {
	Flushable
	Resizable
}

// Runner is in charge of handling the main event loop.  It sets up the screen
// and handles events (input, resizes, etc), delegating out to the views and
// key handlers.
//
// The normal operation works like such:
//
// At any given point in time, the runner has a given View.
//
// When Run starts the main loop, it sets up the screen, and listens for events
// dispatching them as such:
//
// - "Resize" events trigger a resize & redraw of the current view
// - "Update" events populate a new view and redraw
// - "Repaint" events repain the current view
// - "Key" events get sent to the KeyHandler
//
// It's expected that a separate goroutine will receive key events, construct a new
// view based on their operation or based on outside events (like timers for animation,
// new graph data, etc), compute a new view, and send an update.  Think of it kinda like
// a functional reactive UI framework, except a bit inside-out.
type Runner struct {
	screen tcell.Screen
	screenMu sync.Mutex

	// KeyHandler receives key events produced during Run.  It must be specified.
	KeyHandler func(*tcell.EventKey)
	
	// MakeScreen allows custom screens to be used.  Mainly useful for testing.
	// Most cases can use the default value.
	MakeScreen func() (tcell.Screen, error)

	// OnStart is run once the main screen is initialized and the event loop is
	// *about* to start.  Useful for avoiding race conditions regarding the screen
	// being initialized (mainly for the prompt widget & testing).
	OnStart func()
}

// Run initializes the screen, starts the event loop (potentially with an optional
// initial view), and runs it until the given context is closed.  When the context
// is closed, the screen is shut down, and the Run stops.
func (r *Runner) Run(ctx context.Context, initialView View) error {
	var screen tcell.Screen
	if r.MakeScreen == nil {
		var err error
		screen, err = tcell.NewScreen()
		if err != nil {
			return err
		}
	} else {
		var err error
		screen, err = r.MakeScreen()
		if err != nil {
			return err
		}
	}
	screen.Init()
	// TODO(directxman12): we should probably figure out how to call Fini in a
	// defer but before the waiting for the evtLoopDone

	r.screenMu.Lock()
	r.screen = screen
	r.screenMu.Unlock()

	mainView := initialView

	// paint one initial time in case we don't get the immediate resize event
	if mainView != nil {
		mainView.FlushTo(screen)
		screen.Show()
	}

	evtLoopDone := make(chan struct{})
	go func() {
		defer close(evtLoopDone)
		if r.OnStart != nil {
			r.OnStart()
		}
		for evt := screen.PollEvent(); evt != nil; evt = screen.PollEvent() {
			screenCols, screenRows := screen.Size()
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
	screen.Fini()

	// wait till the event loop finishes to actually return this is largely
	// useful for tests to avoid leaking goroutines or accidentally mutating
	// shared state, but technically could be useful if a program was starting
	// and stopping event loops repeatedly
	<-evtLoopDone

	return nil
}

// RequestRepaint requests a repaint of the current view, if any.
// It will not block.
func (r *Runner) RequestRepaint() {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.PostEvent(tcell.NewEventInterrupt(nil))
}

// RequestRepaint replaces the current view & requests a paint of it.
// It will not block.
func (r *Runner) RequestUpdate(newView View) {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.PostEvent(tcell.NewEventInterrupt(newView))
}

// ShowCursor shows the cursor at the given location.
func (r *Runner) ShowCursor(col, row int) {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.ShowCursor(col, row)
}
// HideCursor hides the cursor.
func (r *Runner) HideCursor() {
	r.screenMu.Lock()
	defer r.screenMu.Unlock()

	r.screen.HideCursor()
}
