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
	"context"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/gdamore/tcell"

	"sigs.k8s.io/instrumentation-tools/promq/term"
)

// oneRuneView is a view that writes a single rune ('*' by default) to
// the top left corner of its position box.  It's threadsafe.
type oneRuneView struct {
	pos term.PositionBox
	targetRune rune
	mu sync.Mutex
}
func (v *oneRuneView) FlushTo(screen tcell.Screen) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.targetRune != rune(0) {
		screen.SetContent(v.pos.StartCol, v.pos.StartRow, v.targetRune, nil, tcell.StyleDefault)
	} else {
		screen.SetContent(v.pos.StartCol, v.pos.StartRow, '*', nil, tcell.StyleDefault)
	}
}
func (v *oneRuneView) SetBox(box term.PositionBox) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.pos = box
}
func (v *oneRuneView) GetBox() term.PositionBox {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.pos
}

// threadSafeishScreen is a simulation screen that has a lock over show, so we
// don't race with GetContents in our checkers (which know how to use
// WithScreen).
type threadSafeishScreen struct {
	mu sync.Mutex
	tcell.SimulationScreen
}

// WithScreen provides threadsafe access to the underlying SimulationScreen.
// The SimulationScreen passed to the callback is not valid beyond the body of
// the callback.
func (s *threadSafeishScreen) WithScreen(cb func(tcell.SimulationScreen)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cb(s.SimulationScreen)
}

// Show shows the screen.
func (s *threadSafeishScreen) Show() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SimulationScreen.Show()
}

// Fini marks this as done.
func (s *threadSafeishScreen) Fini() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SimulationScreen.Fini()
}

// waitForPollingStart waits for the runner to start polling, since tcell silently
// drops events until something is polling.
func waitForPollingStart(screen tcell.SimulationScreen, keys <-chan *tcell.EventKey) {
	EventuallyWithOffset(1, func() bool {
		screen.InjectKey(tcell.KeyRune, ' ', tcell.ModNone)
		select {
		case <-keys:
			return true
		default:
			return false
		}
	}).Should(BeTrue())
}
var _ = Describe("The overall Runner", func() {
	var (
		screen *threadSafeishScreen
		cancel context.CancelFunc
		keys chan *tcell.EventKey
		done chan struct{}
		runner *term.Runner
		mainView *oneRuneView = &oneRuneView{}
		initialView term.View
	)
	BeforeEach(func() {
		screen = &threadSafeishScreen{SimulationScreen: tcell.NewSimulationScreen("")}
		initialView = mainView
	})
	JustBeforeEach(func() {
		*mainView = oneRuneView{}

		startedCh := make(chan struct{})
		keys = make(chan *tcell.EventKey, 10 /* some buffer to avoid blocking */)
		localKeys := keys // avoid racing on shutdown, etc
		runner = &term.Runner{
			MakeScreen: func() (tcell.Screen, error) {
				return screen, nil
			},
			KeyHandler: func(key *tcell.EventKey) {
				localKeys <- key
			},
			OnStart: func() {
				close(startedCh)
			},
		}
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())

		// TODO(directxman12): it's fine to run this without a handler, cause we're gonna block
		// till we get an event anyway.  Prob should eventually refactor this
		// test code a bit with a JustBeforeEach or something

		done = make(chan struct{})
		go func() {
			defer GinkgoRecover()
			defer close(done)
			Expect(runner.Run(ctx, initialView)).To(Succeed())
		}()

		// NB(directxman12): events are discarded until we start polling for them,
		// so wait till we're started, send keys until we get a result, then proceed
		<-startedCh
		waitForPollingStart(screen, keys)

		screen.SetSize(10, 10)
	})
	AfterEach(func() {
		cancel()
		<-done // wait till the runner finishes shutting down
	})

	Context("when receiving key events", func() {
		It("should dispatch key events to the key handler", func() {
			screen.InjectKey(tcell.KeyRune, 's', tcell.ModNone)
			screen.InjectKey(tcell.KeyUp, ' ', tcell.ModShift)

			// NB(directxman12): can't just use equal, because there's hidden
			// time fields on the struct
			Eventually(keys).Should(Receive(SatisfyAll(
				WithTransform(func(key *tcell.EventKey) tcell.Key { return key.Key() }, Equal(tcell.KeyRune)),
				WithTransform(func(key *tcell.EventKey) rune { return key.Rune() }, Equal('s')),
				WithTransform(func(key *tcell.EventKey) tcell.ModMask { return key.Modifiers() }, Equal(tcell.ModNone)),
			)))
			Eventually(keys).Should(Receive(SatisfyAll(
				WithTransform(func(key *tcell.EventKey) tcell.Key { return key.Key() }, Equal(tcell.KeyUp)),
				WithTransform(func(key *tcell.EventKey) tcell.ModMask { return key.Modifiers() }, Equal(tcell.ModShift)),
			)))
		})
	})

	It("should switch views when sent a new view", func() {
		Expect(screen).To(DisplayLike(10, 10, "*"))

		runner.RequestUpdate(&oneRuneView{targetRune: '+'})

		Eventually(screen).Should(DisplayLike(10, 10, "+"))
	})

	It("should repaint when a repaint is requested", func() {
		By("manually messing up the screen")
		screen.SetContent(0, 0, 'x', nil, tcell.StyleDefault)
		screen.Show()
		Expect(screen).To(DisplayLike(10, 10, "x"))

		By("requesting a repaint & checking the screen again")
		runner.RequestRepaint()
		Eventually(screen).Should(DisplayLike(10, 10, "*"))
	})

	Context("with no initial view", func() {
		BeforeEach(func() {
			initialView = nil
		})

		It("should skip repainting continue on", func() {
			By("manually messing up the screen")
			screen.SetContent(0, 0, 'x', nil, tcell.StyleDefault)
			screen.Show()
			Expect(screen).To(DisplayLike(10, 10, "x"))

			By("requesting a repaint & checking the screen again")
			runner.RequestRepaint()
			Consistently(screen, "1s").Should(DisplayLike(10, 10, "x"))
		})
	})

	Context("when we get a window resize", func() {
		JustBeforeEach(func() {
			// NB(directxman12): there's tiny bug in SimulationScreen that causes
			// it to decide not send resize events when we call SetSize, so
			// manually inject a resize event here.
			screen.SetSize(12, 13)
			screen.PostEvent(tcell.NewEventResize(12, 13))
		})

		It("should resize the main view", func() {
			Eventually(func() term.PositionBox { return mainView.GetBox() }).Should(Equal(term.PositionBox{Rows: 13, Cols: 12}))
		})

		It("should repaint the main view", func() {
			Eventually(screen).Should(DisplayLike(12, 13, "*"))
		})
	})

	It("should show the cursor when asked to", func() {
		runner.ShowCursor(3, 4)
		col, row, visible := screen.GetCursor()
		Expect(visible).To(BeTrue())
		Expect(col).To(Equal(3))
		Expect(row).To(Equal(4))
	})

	It("should hide the cursor when asked to", func() {
		runner.HideCursor()
		_, _, visible := screen.GetCursor()
		Expect(visible).To(BeFalse())
	})

	Context("when the context is closed", func() {
		It("should shutdown", func() {
			ctx, cancel := context.WithCancel(context.Background())
			runner := &term.Runner{
				MakeScreen: func() (tcell.Screen, error) {
					return tcell.NewSimulationScreen(""), nil
				},
			}
			done := make(chan struct{})

			go func() {
				defer GinkgoRecover()
				defer close(done)
				Expect(runner.Run(ctx, nil)).To(Succeed())
			}()

			cancel()
			Eventually(done).Should(BeClosed())
		})

		PIt("should finalize the screen", func() {
		})
	})
})
