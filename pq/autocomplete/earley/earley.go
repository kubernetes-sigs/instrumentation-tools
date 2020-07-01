/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package earley

import (
	"k8s.io/instrumentation-tools/debug"
)

// Heavily adapted from https://github.com/jakub-m/gearley
// (so much so it's not really the same anymore) (it was really incomplete).
//
// A few key differences: Terminals in this variant of the
// earley parser represent atomic lexical units (which have matched
// some primitive pattern matching algorithm already). The idea
// is that we are traversing our grammar graph until we resolve
// non-terminals with terminals.
//
// This allows us to do some clever things later on, since it is possible
// to make edits to a symbol without changing the lexed token. For instance:
// let's say we have a INT symbol which is [0-9]+ and an operator PLUS '+' and
// we've defined an addition expression which corresponds to INT PLUS INT
//
// Given the string:
//      "1 + 2"
// our lexer returns:
//      [INT, PLUS, INT]
//
// Which we can then feed into our earley parser to generate all possible parses.
//
// Given an edit in the 1 index position, specifically appending an '1' gives
// us the following string:
//      "11 + 2"
// which would also return:
//      [INT, PLUS, INT]
// We do not want to recompute here we know the result of the earley parse already.

const Cursor = "\u25EC"

type Earley struct {
	g     Grammar
	chart *earleyChart
	words Tokens
}

func NewEarleyParser(g Grammar) *Earley {
	newChart := initializeChart(g)
	return &Earley{g: g, chart: newChart}
}

// For every state in S(k) of the form (X → α • Y β, j)
// (where j is the origin position as above), add (Y → • γ, k) to S(k)
// for every production in the grammar with Y on the left-hand side (Y → γ).
func (p *Earley) predict(state *EarleyItem, chartIndex int) {
	nextSymbol := state.GetRightSymbolByRulePos().(nonTerminal)
	recognizedRules := p.g.recognizedRules(nextSymbol)
	currStateSet := p.chart.GetState(chartIndex)
	if len(recognizedRules) > 0 {
		debug.Debugf("Predicting next state\n")
	}
	// Find all the rules for the Symbol put those rules to the current set
	for _, r := range recognizedRules {
		nextItem := newPredictItem(r, chartIndex, state.ctx)
		if currStateSet.Add(nextItem) {
			debug.Debugf("added %v\n", nextItem.String())
		}
	}
}

// If a is the next symbol in the input stream, for every state in S(k) of the
// form (X → α • a β, j), add (X → α a • β, j) to S(k+1).
func (p *Earley) scan(state *EarleyItem, chartIndex int, token Tokhan) {
	// abort though if we can't scan further
	if chartIndex+1 >= p.chart.Length() || !state.DoesTokenTypeMatch(token) {
		return
	}
	debug.Debugf("Token (%v) matches, performing scan \n", token)
	ctx := &completionContext{}
	if state.ctx != nil {
		ctx = state.ctx
	}
	ctx.BuildContext(state.GetRightSymbolTypeByRulePos(), &token)

	nextItem := newScanItem(state, chartIndex, ctx)
	debug.Debugf("Scanning next item : %v\n", nextItem)
	// scanned item is added to next stateSet
	nextSet := p.chart.GetState(chartIndex + 1)
	nextSet.Add(nextItem)
}

// For every state in S(k) of the form (Y → γ •, j), find all states in S(j)
// of the form (X → α • Y β, i) and add (X → α Y • β, i) to S(k).
func (p *Earley) complete(state *EarleyItem, chartIndex int) {
	currStateSet := p.chart.GetState(chartIndex)
	originalSet := p.chart.GetState(state.originatingIndex)
	itemsToComplete := originalSet.findItemsToComplete(state.Rule.left)

	for _, item := range itemsToComplete {
		nextItem := newCompleteItem(&item)
		if currStateSet.Add(nextItem) {
			debug.Debugf("completed %v\n", nextItem.String())
		}
	}
}

func (p *Earley) resizeChartIfNecessary(words Tokens) {
	p.resizeChart(len(words) + 1)
}

func (p *Earley) resizeChart(size int) {
	currentSize := p.chart.Length()
	for currentSize < size {
		p.chart.append(NewStateSet())
		currentSize += 1
	}
}

// Parse parses the full input string. It first tokenizes the input
// string and then uses those tokens as atomic units in our
// grammar, which simplifies our parsing logic considerably.
func (p *Earley) Parse(input string) *earleyChart {
	inputWords := extractWords(input)
	return p.ParseTokens(inputWords)
}

// Parse parses the full input string. It first tokenizes the input
// string and then uses those tokens as atomic units in our
// grammar, which simplifies our parsing logic considerably.
func (p *Earley) ParseTokens(tokens Tokens) *earleyChart {
	p.words = tokens
	p.resizeChartIfNecessary(tokens)
	// parse through all words
	for stateIndex := 0; stateIndex <= len(tokens); stateIndex++ {
		p.PartialParse(tokens, stateIndex)
		debug.Debugf("------\n%v\n------\n", p.chart.String())
	}
	return p.chart
}

// This is the incremental bit of our parser. We can basically feed in
// a list of words (i.e. lexed tokens) and parse at a specific word index.
func (p *Earley) PartialParse(tokens Tokens, chartIndex int) *StateSet {
	debug.Debugf("------- starting partial parse %v\n", chartIndex)
	p.words = tokens
	// we're going to assume here that we've correctly parsed prior to our index
	p.resizeChartIfNecessary(tokens)
	currStateSet := p.chart.GetState(chartIndex)
	for _, token := range tokens {
		setIndex := 0
		for {
			if setIndex >= len(currStateSet.GetStates()) {
				break
			}
			state := currStateSet.items[setIndex]
			if !state.isCompleted() {
				// predict if current state isn't terminal
				if !state.GetRightSymbolByRulePos().isTerminal() {
					p.predict(state, chartIndex)
				} else {
					// Scan the next symbol which is terminal
					p.scan(state, chartIndex, token)
				}
			} else { // end of rule, let's complete
				p.complete(state, chartIndex)
			}
			setIndex++
		}
	}
	return currStateSet
}

func (p *Earley) GetSuggestedTokenType(tokens Tokens) (types []ContextualToken) {
	lastTokenPos := len(tokens) - 1
	if lastTokenPos < 0 {
		lastTokenPos = 0
	}
	p.ParseTokens(tokens)
	suggestions := p.chart.GetValidTerminalTypesAtStateSet(lastTokenPos)
	debug.Debugln(
		"generating suggestions", tokens.Vals()[lastTokenPos], len(tokens), lastTokenPos, len(suggestions))
	return suggestions
}
