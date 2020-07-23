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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/gdamore/tcell"
	goprompt "github.com/c-bata/go-prompt"

	"sigs.k8s.io/instrumentation-tools/promq/term"
)

// fakeScreenish wraps a simulationScreen to allow us to quickly test the
// prompt widget w/o using a whole runner.
type fakeScreenish struct {
	screen tcell.SimulationScreen
	view term.Flushable
}
func (s *fakeScreenish) ShowCursor(col, row int) {
	s.screen.ShowCursor(col, row)
}
func (s *fakeScreenish) HideCursor() {
	s.screen.HideCursor()
}
func (s *fakeScreenish) RequestRepaint() {
	s.view.FlushTo(s.screen)
	s.screen.Show()
}

// sendASCIIKeys sends strings with keypresses in the ascii range
func sendASCIIKeys(str string, pr *term.PromptView) {
	for _, rn := range str {
		pr.HandleKey(tcell.NewEventKey(tcell.KeyRune, rn, tcell.ModNone))
	}
}


var _ = Describe("The Prompt widget", func() {
	var (
		screen tcell.SimulationScreen
		prompt *term.PromptView
		waitForSetup chan struct{}

		testCompleter = func(d goprompt.Document) []goprompt.Suggest {
			return goprompt.FilterHasPrefix([]goprompt.Suggest{
				{Text: "cheddar", Description: "only sharp cheddars allowed"},
				{Text: "parmesan", Description: "you'd better not mention the powdered stuff"},
				{Text: "pepper jack", Description: "mmm... spicy"},
			}, d.GetWordBeforeCursor(), true)
		}

		ctx context.Context
		cancel context.CancelFunc
	)
	BeforeEach(func() {
		screen = tcell.NewSimulationScreen("")
		screen.Init()
		screen.SetSize(50, 10)
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

		prompt.Screen = &fakeScreenish{
			screen: screen,
			view: prompt,
		}

		prompt.SetBox(term.PositionBox{Rows: 10, Cols: 50})

		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	})
	AfterEach(func() {
		screen.Fini()
		cancel()
	})

	It("should translate key events into key presses on the screen", func() {
		go func() {
			defer GinkgoRecover()
			Expect(prompt.Run(ctx, nil, cancel)).To(Succeed())
		}()

		// wait for setup so that it's safe to send keypresses
		<-waitForSetup

		sendASCIIKeys("ch", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			"> ch                                              "+
			"     cheddar  only sharp cheddars allowed         ",
		))

		sendASCIIKeys("edda", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			"> chedda                                          "+
			"         cheddar  only sharp cheddars allowed     ",
		))

		sendASCIIKeys("\t", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			"> cheddar                                         "+
			"         cheddar  only sharp cheddars allowed     ",
		))

		sendASCIIKeys("\r", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			"> cheddar                                         ",
		))
	})

	It("should handle using non-displayable keys to navigate the completions", func() {
		go func() {
			defer GinkgoRecover()
			Expect(prompt.Run(ctx, nil, cancel)).To(Succeed())
		}()

		// wait for setup so that it's safe to send keypresses
		<-waitForSetup

		sendASCIIKeys("\t", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			">                                                 "+
			"   cheddar      only sharp cheddars allowed       "+
			"   parmesan     you'd better not mention the ...  "+
			"   pepper jack  mmm... spicy                      ",
		))

		prompt.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
		Eventually(screen).Should(DisplayLike(50, 10,
			">                                                 "+
			"   cheddar      only sharp cheddars allowed       "+
			"   parmesan     you'd better not mention the ...  "+
			"   pepper jack  mmm... spicy                      ",
		))

		sendASCIIKeys("\t", prompt)
		Eventually(screen).Should(DisplayLike(50, 10,
			"> cheddar                                         "+
			"   cheddar      only sharp cheddars allowed       "+
			"   parmesan     you'd better not mention the ...  "+
			"   pepper jack  mmm... spicy                      ",
		))

		prompt.HandleKey(tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone))
		Eventually(screen).Should(DisplayLike(50, 10,
			"> parmesan                                        "+
			"   cheddar      only sharp cheddars allowed       "+
			"   parmesan     you'd better not mention the ...  "+
			"   pepper jack  mmm... spicy                      ",
		))
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

	PContext("when rendering", func() {
		It("should start rendering at its box's position", func() {

		})

		It("should never render outside its box", func() {

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

			go func() {
				defer GinkgoRecover()
				Expect(prompt.Run(ctx, nil, cancel)).To(Succeed())
			}()

			// wait for setup so that it's safe to send keypresses
			<-waitForSetup

			sendASCIIKeys("parmesean\r", prompt)

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
			prompt.FlushTo(screen)
			screen.Show()

			Expect(screen).Should(DisplayLike(50, 10,
				">                                                 "+
				"now to find some crackers to go with that!",
			))
		})

		It("it should not display a blank line between prompts if no output is given", func() {
			cnt := 0
			prompt.HandleInput = func(_ string) (text *string, stop bool) {
				cnt++
				if cnt == 1 {
					return nil, false
				}
				return nil, true
			}

			go func() {
				defer GinkgoRecover()
				Expect(prompt.Run(ctx, nil, cancel)).To(Succeed())
			}()
			// wait for setup so that it's safe to send keypresses
			<-waitForSetup

			sendASCIIKeys("cheddar\r", prompt)
			sendASCIIKeys("monterey jack\r", prompt)

			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"> monterey jack                                   ",
			))
		})

		It("should continue presenting prompts & output until asked to stop, then shutdown the screen", func() {
			cnt := 0
			prompt.HandleInput = func(_ string) (text *string, stop bool) {
				cnt++
				txt := fmt.Sprintf("count: %d\n", cnt)
				if cnt < 3 {
					return &txt, false
				}
				return &txt, true
			}

			go func() {
				defer GinkgoRecover()
				Expect(prompt.Run(ctx, nil, cancel)).To(Succeed())
			}()
			// wait for setup so that it's safe to send keypresses
			<-waitForSetup

			sendASCIIKeys("cheddar\r", prompt)
			sendASCIIKeys("monterey jack\r", prompt)
			sendASCIIKeys("parmesean\r", prompt)


			Eventually(screen).Should(DisplayLike(50, 10,
				"> cheddar                                         "+
				"count: 1                                          "+
				"> monterey jack                                   "+
				"count: 2                                          "+
				"> parmesean                                       ",
				// normally, the main loop handles the last draw, so skip the final output
			))
		})
	})
})
