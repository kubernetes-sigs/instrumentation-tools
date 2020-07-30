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
	"context"
	"errors"
	"github.com/c-bata/go-prompt"
	"github.com/gdamore/tcell"
)

// PromptView is a widget that displays a go-prompt prompt.  It feeds input
// from HandleKey to go-prompt and writes output to an internal buffer that can
// later be flushed to the screen as expected.  Since go-prompt is written
// around having some control of the screen, unlike most widgets this one
// should be "persistent" across updates, and the internals go to some lengths
// to keep things threadsafe.
//
// Due to the internals, this widget shouldn't be used until Run is called and
// you've received a callback to OnSetup.
type PromptView struct {
	writer *cellWriter

	reader *screenParser

	Screen screenIsh
	start chan struct{}

	pos PositionBox

	// SetupPrompt initializes the go-prompt prompt each time it requests new
	// input.
	SetupPrompt func(requiredOpts ...prompt.Option) *prompt.Prompt
	// HandleInput is called when the prompt returns with an entered command or whatnot.
	// Returned text is written before displaying a new prompt, and stop can be used to
	// indicate that even event loop should be stopped and the screen shut down (e.g. an
	// exit command).
	HandleInput func(input string) (text *string, stop bool)
	// OnSetup is called during Run once the reader and writer are initialized.  It's
	// useful for avoiding races when adding this to be displayed -- you shouldn't try
	// to use this for displaying until OnSetup has been called.
	OnSetup func()
}

func (v *PromptView) SetBox(box PositionBox) {
	v.pos = box
	if v.reader != nil && v.writer != nil {
		v.writer.SetBox(box)
		v.reader.Resize(&prompt.WinSize{Row: uint16(box.Rows), Col: uint16(box.Cols)})
	}

	if v.start != nil {
		close(v.start)
		v.start = nil
	}
}
func (v *PromptView) FlushTo(screen tcell.Screen) {
	v.writer.FlushTo(screen, v.pos.StartCol, v.pos.StartRow)
}

// HandleKey receives key events from an input loop.  Use this as the runner's
// KeyHandler.
func (v *PromptView) HandleKey(evt *tcell.EventKey) {
	v.reader.AddKey(evt)
}

// Run starts the go-prompt even loop, repeatedly asking the user for input, handling the input
// and dispatching the result to HandleInput.  If given, initial input is prepopulated into the
// prompt followed by a synthetic "enter" event.  shutdownScreen will be called when HandleInput
// asks for to be stopped.  It's generally the Runner's context's cancel function.
func (v *PromptView) Run(ctx context.Context, initialInput *string, shutdownScreen func()) error {
	v.writer = &cellWriter{
		screen: v.Screen,
	}
	v.reader = &screenParser{
		evts: make(chan *tcell.EventKey, 30),
	}
	viewPrompt := v.SetupPrompt(prompt.OptionParser(v.reader), prompt.OptionWriter(v.writer))
	start := make(chan struct{})
	v.start = start

	// we've already been asked to start
	if v.pos != (PositionBox{}) {
		v.SetBox(v.pos)
	}

	if v.OnSetup != nil {
		v.OnSetup()
	}

	go func() {
		<-start
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// grumble grumple can't handle resize cleanly without also
			// allowing go-prompt to call os.Exit grumble grumble
			input := viewPrompt.Input()
			output, stop := v.HandleInput(input)
			if output != nil {
				v.writer.WriteStr(*output)
				v.writer.Flush() // always flush after we write
			}
			if stop {
				shutdownScreen()
				return
			}
		}
	}()

	// do this after starting the input loop so that we don't block on the channel capacity
	if initialInput != nil {
		// populate the initial command
		v.reader.AddString(*initialInput+"\r")
	}

	<-ctx.Done()

	if err := ctx.Err(); !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}
