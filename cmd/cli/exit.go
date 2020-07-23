/*
Copyright 2019 The Kubernetes Authors.

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

package cli

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/sets"
	"math/rand"
	"os"
	"strings"
	"time"
)

var (
	exitStrings = sets.NewString("q", "quit", "exit")

	exitQuotes = []string{
		"\nPeople say nothing is impossible, but I do nothing every day.",
		"\nI want my children to have all the things I couldn’t afford. Then I want to move in with them.",
		"\nI have always wanted to be somebody, but I see now I should have been more specific.",
		"\nI finally realized that people are prisoners of their phones... that's why it's called a 'cell' phone.",
		"\nSometimes when I close my eyes, I can't see.",
		"\nHere’s some advice: At a job interview, tell them you’re willing to give 110 percent. Unless the job is a statistician.",
		"\nWhy do they call it rush hour when nothing moves?",
		"\nNever put off till tomorrow what you can do the day after tomorrow just as well.",
	}
)

func ExitFunc(qs string) bool {
	qs = strings.TrimSpace(qs)
	if qs == "" {
		return false
	} else if exitStrings.Has(qs) {
		s := rand.NewSource(time.Now().Unix())
		r := rand.New(s)
		fmt.Println(exitQuotes[r.Intn(len(exitQuotes))])
		os.Exit(0)
		return true
	}
	return false
}
