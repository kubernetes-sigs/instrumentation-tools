/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
    "github.com/c-bata/go-prompt"

	"sigs.k8s.io/instrumentation-tools/promq/autocomplete"
)

const (
	// spaces can't individually demarcate individual lexical units
	// in promql.
	PromQLTokenSeparators = " []{}()=!~,"
)

type Completer struct {
	promCompleter autocomplete.PromQLCompleter
}

func NewCompleter(completor autocomplete.PromQLCompleter) *Completer {
	return &Completer{promCompleter: completor}
}

func (c *Completer) Complete(d prompt.Document) []prompt.Suggest {
	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}
	ret := c.promCompleter.GenerateSuggestions(d.Text, d.DisplayCursorPosition())
	suggests := make([]prompt.Suggest, len(ret))
	for i, s := range ret {
		suggests[i] = prompt.Suggest{Text: s.GetValue(), Description: s.GetDetail()}
	}
	return suggests
}
