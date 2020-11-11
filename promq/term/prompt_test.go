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
	"time"
	"fmt"
	"sync"
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/gdamore/tcell"
	goprompt "github.com/c-bata/go-prompt"

	"sigs.k8s.io/instrumentation-tools/promq/term"
)

// fakeScreenish wraps a simulationScreen to allow us to quickly test the
// prompt widget w/o using a whole runner.  It's threadsafe, unlike SimulationScreen.
// Use WithScreen if you want threadsafe access to the underlying screen.
type fakeScreenish struct {
	mu sync.Mutex
	screen tcell.SimulationScreen
	view term.Flushable
}
func (s *fakeScreenish) ShowCursor(col, row int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.screen.ShowCursor(col, row)
}
func (s *fakeScreenish) HideCursor() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.screen.HideCursor()
}
func (s *fakeScreenish) RequestRepaint() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.view.FlushTo(s.screen)
	s.screen.Show()
}
// WithScreen provides threadsafe access to the underlying SimulationScreen.
// The SimulationScreen passed to the callback is not valid beyond the body of
// the callback.
func (s *fakeScreenish) WithScreen(cb func(tcell.SimulationScreen)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cb(s.screen)
}

// sendRuneKeys sends strings with keypresses that consist of single-rune
// sequences (e.g. no control characters, non-rune keys, combining characters,
// etc).
func sendRuneKeys(str string, pr *term.PromptView) {
	for _, rn := range str {
		pr.HandleKey(tcell.NewEventKey(tcell.KeyRune, rn, tcell.ModNone))
	}
}

var (
	testCompleter = func(d goprompt.Document) []goprompt.Suggest {
		return goprompt.FilterHasPrefix([]goprompt.Suggest{
			{Text: "cheddar", Description: "only sharp cheddars allowed"},
			{Text: "parmesan", Description: "you'd better not mention the powdered stuff"},
			{Text: "pepper jack", Description: "mmm... spicy"},
		}, d.GetWordBeforeCursor(), true)
	}
)

var _ = Describe("The Prompt widget", func() {
	var (
		screen *fakeScreenish
		prompt *term.PromptView
		waitForSetup chan struct{}

		ctx context.Context
		cancel context.CancelFunc
	)
	BeforeEach(func() {
		if _, err := os.Open("/dev/tty"); err != nil {
			Skip("there's a weird bug in go-prompt where it always tries to initialize a tty parser, so skip if we don't have one")
		}

		rawScreen := tcell.NewSimulationScreen("")
		rawScreen.Init()
		rawScreen.SetSize(50, 10)
		waitForSetup = make(chan struct{})

		prompt = &term.PromptView{
			HandleInput: func(input string) (text *string, stop bool) {
				// by default, stop immediately once stuff is entered
				return nil, true
			},
			SetupPrompt: func(requiredOpts ...goprompt.Option) *goprompt.Prompt {
				return goprompt.New(nil, testCompleter, requiredOpts...)
			},
			OnSetup: func() {
				close(waitForSetup)
			},
		}

		screen = &fakeScreenish{
			screen: rawScreen,
			view: prompt,
		}
		prompt.Screen = screen

		prompt.SetBox(term.PositionBox{Rows: 10, Cols: 50})

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	})

	Context("when translating key events into key presses", func() {
		var waitForPrompt *promptWaiter

		BeforeEach(func() {
			waitForPrompt = runPromptInBg(ctx, prompt)
			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")
		})

		AfterEach(func() {
			waitForPrompt.WaitTillDone()
		})

		It("should map single-byte rune key presses to key presses in the prompt, and display them", func() {

			sendRuneKeys("ch", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> ch                                              "+
				"     cheddar  only sharp cheddars allowed         ",
			))

			sendRuneKeys("edda", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> chedda                                          "+
				"         cheddar  only sharp cheddars allowed     ",
			))

			sendRuneKeys("\t", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"         cheddar  only sharp cheddars allowed     ",
			))

			sendRuneKeys("\r", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         ",
			))
		})

		It("should map multi-byte rune key presses to key presses in the prompt, and display them", func() {
			sendRuneKeys("warning sign: \u26a0", prompt)
			Eventually(screen).Should(DisplayLike(50, 10, "> warning sign: \u26a0"))
		})

		It("should map common non-rune key events to keypresses used to nativate completions", func() {
			// TODO(directxman12): these should prob be separate tests

			By("opening the completions menu with tab")
			sendRuneKeys("\t", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				">                                                 "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			By("selecting a completion with down arrow & completing it with tab")
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
			Eventually(screen).Should(DisplayLike(50, 10,
				">                                                 "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			sendRuneKeys("\t", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			By("selecting a different completion with down arrow")
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
			Eventually(screen).Should(DisplayLike(50, 10,
				"> parmesan                                        "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			By("going back with up arrow")
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone))
			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			By("navigating left with left arrow")
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyLeft, 0, tcell.ModNone))
			sendRuneKeys(" ", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> chedd ar                                        "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))

			By("navigating right with right arrow")
			prompt.HandleKey(tcell.NewEventKey(tcell.KeyRight, 0, tcell.ModNone))
			sendRuneKeys(" ", prompt)
			Eventually(screen).Should(DisplayLike(50, 10,
				"> chedd a r                                       "+
				"   cheddar      only sharp cheddars allowed       "+
				"   parmesan     you'd better not mention the ...  "+
				"   pepper jack  mmm... spicy                      ",
			))
		})
	})


	It("should populate initial input into the prompt if given", func() {
		initialInput := "pepper jack"
		Expect(prompt.Run(ctx, &initialInput, cancel)).To(Succeed())

		Eventually(screen).Should(DisplayLike(50, 10, "> pepper jack"))
	})

	It("should shutdown when the context is closed", func() {
		doneCh := make(chan struct{})
		go func() {
			defer GinkgoRecover()
			Expect(prompt.Run(ctx, nil, func() {})).To(Succeed())
			close(doneCh)
		}()

		cancel()
		Eventually(doneCh).Should(BeClosed())
	})

	Context("when rendering", func() {
		It("should start rendering at its box's position", func() {
			prompt.SetBox(term.PositionBox{StartRow: 1, StartCol: 2, Rows: 7, Cols: 45})

			defer runPromptInBg(ctx, prompt).WaitTillDone()

			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")

			Eventually(screen).Should(DisplayLike(50, 10,
				"                                                  "+
				"  >                                               ",
			))
		})

		It("should never render outside its box", func() {
			Skip("this has a weird race that's making it flaky -- see note below")

			// go-prompt eagerly consumes input then discards anything after
			// the first \r, so wait till we get a new prompt between inputs
			// like a real user would
			pulse := make(chan struct{}, 10) 
			prompt.HandleInput = func(_ string) (*string, bool) {
				// never exit normally, just use the cancel in aftereach
				pulse <- struct{}{}
				return nil, false
			}

			defer runPromptInBg(ctx, prompt).WaitTillDone()

			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")

			By("using a box smaller than the screen box & marking the end manually")
			prompt.SetBox(term.PositionBox{Cols: 40, Rows: 7})
			screen.WithScreen(func(screen tcell.SimulationScreen) {
				screen.SetContent(0, 7, '*', nil, tcell.StyleDefault)
				screen.SetContent(45, 0, '*', nil, tcell.StyleDefault)
			})

			By("entering in a few lines of text, and checking that we scrolled")
			sendRuneKeys("cheddar\r", prompt)
			<-pulse
			sendRuneKeys("pepper jack\r", prompt)
			<-pulse
			sendRuneKeys("monterey jack\r", prompt)
			<-pulse
			sendRuneKeys("wensleydale\r", prompt)
			// TODO(directxman12): there's a *really* weird race here around
			// the displaying of the old two lines (monterey jack &
			// wensleydale).  I'm not sure why or how to correctly reproduce :-/
			// Set this to skip if it becomes a problem.
			Eventually(screen).Should(DisplayLike(50, 10,
				"> monterey jack                              *    "+
				"> wensleydale                                     "+
				">                                                 "+
				"   cheddar      only sharp cheddars...            "+
				"   parmesan     you'd better not me...            "+
				"   pepper jack  mmm... spicy                      "+
				"                                                  "+
				"*                                                 ",
			))
		})
	})

	It("should allow configuring prompt setup via a callback", func() {
		prompt.SetupPrompt = func(reqOpts ...goprompt.Option) *goprompt.Prompt {
			opts := append([]goprompt.Option(nil), reqOpts...)
			opts = append(opts, goprompt.OptionPrefix("type of cheese> "))

			return goprompt.New(nil, testCompleter, opts...)
		}

		initialInput := "" // just send some input so we only do one cycle
		Expect(prompt.Run(ctx, &initialInput, cancel)).To(Succeed())

		Eventually(screen).Should(DisplayLike(50, 10, "type of cheese> "))
	})

	Context("when handling results", func() {
		It("should dispatch succesful input to the given callback", func() {
			textCh := make(chan string, 1 /* buffer so we don't block */)
			defer close(textCh)

			prompt.HandleInput = func(input string) (text *string, stop bool) {
				textCh <- input
				return nil, true
			}

			defer runPromptInBg(ctx, prompt).WaitTillDone()

			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")

			sendRuneKeys("parmesean\r", prompt)

			Eventually(textCh).Should(Receive(Equal("parmesean")))
		})

		It("should display output from the results handler, if present", func() {
			prompt.HandleInput = func(_ string) (text *string, stop bool) {
				output := "now to find some crackers to go with that!"
				return &output, true
			}

			initialInput := ""
			Expect(prompt.Run(ctx, &initialInput, cancel)).To(Succeed())

			// normally the main loop would handle the last draw after shutdown
			screen.WithScreen(func(screen tcell.SimulationScreen) {
				prompt.FlushTo(screen)
				screen.Show()
			})

			Expect(screen).Should(DisplayLike(50, 10,
				">                                                 "+
				"now to find some crackers to go with that!",
			))
		})

		It("it should not display a blank line between prompts if no output is given", func() {
			cnt := 0
			pulse := make(chan struct{}, 10) 
			prompt.HandleInput = func(_ string) (text *string, stop bool) {
				cnt++
				pulse <- struct{}{}
				if cnt == 1 {
					return nil, false
				}
				return nil, true
			}

			defer runPromptInBg(ctx, prompt).WaitTillDone()

			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")

			sendRuneKeys("cheddar\r", prompt)
			<-pulse
			sendRuneKeys("monterey jack\r", prompt)

			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"> monterey jack                                   ",
			))
		})

		It("should continue presenting prompts & output until asked to stop, then shutdown the screen", func() {
			// go-prompt eagerly consumes input then discards anything after
			// the first \r, so wait till we get a new prompt between inputs
			// like a real user would
			pulse := make(chan struct{}, 10) 
			cnt := 0
			prompt.HandleInput = func(_ string) (text *string, stop bool) {
				cnt++
				txt := fmt.Sprintf("count: %d\n", cnt)
				pulse <- struct{}{}
				if cnt < 3 {
					return &txt, false
				}
				return &txt, true
			}

			defer runPromptInBg(ctx, prompt).WaitTillDone()
			// wait for setup so that it's safe to send keypresses
			Eventually(waitForSetup).Should(BeClosed(), "should be safe to send keypresses eventually")

			sendRuneKeys("cheddar\r", prompt)
			<-pulse
			sendRuneKeys("monterey jack\r", prompt)
			<-pulse
			sendRuneKeys("parmesean\r", prompt)
			<-pulse
			sendRuneKeys("\r", prompt)

			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"count: 1                                          "+
				"> monterey jack                                   "+
				"count: 2                                          "+
				"> parmesean                                       "+
				"count: 3                                          ",
			))
		})
	})
})

func runPromptInBg(ctx context.Context, prompt *term.PromptView) *promptWaiter {
	ctx, cancel := context.WithCancel(ctx)

	doneCh := make(chan struct{})
	go func() {
		defer GinkgoRecover()
		defer close(doneCh)
		ExpectWithOffset(1, prompt.Run(ctx, nil, cancel)).To(Succeed())
	}()

	return &promptWaiter{
		doneCh: doneCh,
		cancel: cancel,
	}
}

type promptWaiter struct {
	doneCh <-chan struct{}
	cancel context.CancelFunc
}

func (w *promptWaiter) WaitTillDone() {
	w.cancel()
	EventuallyWithOffset(1, w.doneCh).Should(BeClosed())
}
