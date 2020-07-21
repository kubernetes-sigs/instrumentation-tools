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

package earley

import (
	"fmt"
	"strings"
	"testing"
)

var (
	P = NewNonTerminal("P", true)

	S = NewNonTerminal("S", false)
	M = NewNonTerminal("M", false)
	N = NewNonTerminal("N", false)
	T = NewNonTerminal("T", false)

	a     = NewTerminal("a")
	b     = NewTerminal("b")
	plus  = NewTerminal("+")
	multi = NewTerminal("*")

	testGrammar = newTestGrammar()
)

func newTestGrammar() *Grammar {
	return NewGrammar(
		// P -> S
		NewRule(P, S),
		// S -> S + M
		NewRule(S, S, plus, M),
		// S -> M
		NewRule(S, M),
		// M -> M * T
		NewRule(M, M, multi, T),
		// M -> T
		NewRule(M, T),
		// T -> a
		NewRule(T, a),
		// T -> b
		NewRule(T, b),
	)
}

func TestEarleyStates(t *testing.T) {
	testCases := []struct {
		name string
	}{
		{
			name: "first test",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			st := PromQLParser.Parse("-1")
			fmt.Println(st)
			if st.String() == "" {
				t.Errorf("Did not expect this to serialize into empty string")
			}
		})
	}
}

func TestEarleyChartStates(t *testing.T) {
	testCases := []struct {
		tokenType             string
		inputString           string
		expectedTokenCount    int
		expectedStatesAtIndex []*StateSet
	}{
		{
			tokenType:             "Test we have our expected states",
			inputString:           "sum(metric{label='value'}) by (label)",
			expectedTokenCount:    7,
			expectedStatesAtIndex: make([]*StateSet, 7),
		},
	}
	for _, tc := range testCases {
		p := NewEarleyParser(*promQLGrammar)
		inputWords := extractWords(tc.inputString)
		p.words = inputWords
		p.resizeChartIfNecessary(inputWords)
		t.Logf("%v\n", len(p.words))

		for i := 0; i < len(p.words); i++ {
			t.Run(fmt.Sprintf("%v", i), func(t *testing.T) {

				//gotSS := p.PartialParse(inputWords, i)
				//if gotSS != tc.expectedStatesAtIndex[i] {
				//	t.Errorf("Got \n%v\n", gotSS)
				//}
			})
		}
	}
}

func TestEarleyPredict(t *testing.T) {
	testCases := []struct {
		name             string
		item             *EarleyItem
		chartIndex       int
		expectedNewItems []string
	}{
		{
			name: "Add predict items",
			item: &EarleyItem{
				// P -> S
				Rule:  testGrammar.rules[0],
				cause: "predict",
			},
			chartIndex: 0,
			expectedNewItems: []string{
				"Rule(S) ->  ◬ M (0) (cause:predict) (tokensConsumed:0)",
				"Rule(S) ->  ◬ S '+' M (0) (cause:predict) (tokensConsumed:0)",
			},
		},
		{
			name: "Item has already been in the state set",
			item: &EarleyItem{
				// S -> S + M
				Rule:  testGrammar.rules[1],
				cause: "predict",
			},
			chartIndex:       0,
			expectedNewItems: []string{},
		},
	}

	parser := NewEarleyParser(*testGrammar)
	for _, tc := range testCases {
		stateSet := parser.chart.GetState(tc.chartIndex)
		prevItemLength := len(stateSet.items)
		parser.predict(tc.item, tc.chartIndex, nil)
		verifyStateSet(t, stateSet, tc.expectedNewItems, prevItemLength)
	}

}

func TestEarleyScan(t *testing.T) {
	testCases := []struct {
		name             string
		item             *EarleyItem
		token            Tokhan
		chartIndex       int
		chartSize        int
		expectedNewItems []string
	}{
		{
			name:             "Index out of bound",
			item:             nil,
			token:            Tokhan{Type: "b"},
			chartIndex:       1,
			chartSize:        2,
			expectedNewItems: []string{},
		},
		{
			name: "Token type not match",
			item: &EarleyItem{
				// T -> a
				Rule:  testGrammar.rules[5],
				cause: PREDICT_STATE,
			},
			token:            Tokhan{Type: "b"},
			chartIndex:       0,
			chartSize:        2,
			expectedNewItems: []string{},
		},
		{
			name: "Add scan items",
			item: &EarleyItem{
				// T -> a
				Rule:  testGrammar.rules[5],
				cause: PREDICT_STATE,
			},
			token:            Tokhan{Type: "a"},
			chartIndex:       0,
			chartSize:        2,
			expectedNewItems: []string{"Rule(T) -> 'a' ◬  (0) (cause:scan) (tokensConsumed:1)"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewEarleyParser(*testGrammar)
			parser.resizeChart(tc.chartSize)

			stateSet := parser.chart.GetState(1)
			prevItemLength := len(stateSet.items)
			parser.scan(tc.item, tc.chartIndex, nil, tc.token)
			verifyStateSet(t, stateSet, tc.expectedNewItems, prevItemLength)
		})
	}

}

func TestEarleyComplete(t *testing.T) {
	testCases := []struct {
		name            string
		item            *EarleyItem
		chartIndex      int
		expectedNewItem []string
	}{
		{
			name: "Add complete items",
			item: &EarleyItem{
				Rule:             testGrammar.rules[5],
				RulePos:          1,
				originatingIndex: 0,
				cause:            SCAN_STATE,
			},
			chartIndex:      1,
			expectedNewItem: []string{"Rule(M) -> T ◬  (0) (cause:complete) (tokensConsumed:0)"},
		},
		{
			name: "Item has already been in the state set",
			item: &EarleyItem{
				Rule:             testGrammar.rules[4],
				RulePos:          1,
				originatingIndex: 0,
				cause:            COMPLETE_STATE,
			},
			chartIndex:      1,
			expectedNewItem: []string{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewEarleyParser(*testGrammar)
			parser.resizeChart(tc.chartIndex + 1)

			chart := parser.chart
			set0 := chart.GetState(0)
			// Add predict item M -> ◬ T to set 0
			set0.items = append(set0.items, newPredictItem(testGrammar.rules[4], 0, nil, nil))
			set1 := chart.GetState(1)
			// Add complete item M -> M ◬ * T to set 1
			set1.items = append(set1.items, &EarleyItem{
				Rule:             testGrammar.rules[3],
				RulePos:          1,
				originatingIndex: 0,
				cause:            COMPLETE_STATE,
			})
			// Add complete item S -> M ◬ to set 1
			set1.items = append(set1.items, &EarleyItem{
				Rule:             testGrammar.rules[2],
				RulePos:          1,
				originatingIndex: 0,
				cause:            COMPLETE_STATE,
			})
			stateSet := chart.GetState(tc.chartIndex)
			prevItemLength := len(stateSet.items)
			parser.complete(tc.item, tc.chartIndex)
			verifyStateSet(t, stateSet, tc.expectedNewItem, prevItemLength)
		})
	}

}

// verify stateset value and length
func verifyStateSet(t *testing.T, stateSet *StateSet, expected []string, prevLength int) {
	for _, str := range expected {
		find := false
		for _, item := range stateSet.items {
			t.Log(item.String())
			if strings.Contains(item.String(), str) {
				find = true
				break
			}
		}
		if !find {
			t.Errorf("expected to have item %v in stateSet but not", str)
		}
	}
	if len(stateSet.items) != prevLength+len(expected) {
		t.Errorf("expected length of stateSet is %d, got %d",
			prevLength+len(expected), len(stateSet.items))
	}
}
