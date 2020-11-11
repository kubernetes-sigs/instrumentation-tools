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
)

// EarleyItem represents A SINGLE possible parse path. More abstractly,
// this represents a potential grammar rule which we can validly apply.
// It is the basic unit of state set.
//noinspection GoNameStartsWithPackageName
type EarleyItem struct {
	// id is the position of this item in the chart; e.g. [1,0] is the first item in Stateset 1
	id      ItemId
	Rule    *GrammarRule
	RulePos int // dot position
	// the position in the input at which the matching of the production began
	originatingIndex int
	cause            StateType // 'predict', 'scan' or 'complete'
	// from is an array of the existing item that generate this item
	from                    []ItemId
	terminalSymbolsConsumed int
	ctx                     *completionContext
}

type ItemId struct {
	StateSetIndex int
	ItemIndex     int
}

func newPredictItem(r *GrammarRule, index int, from []ItemId, ctx *completionContext) *EarleyItem {
	return &EarleyItem{
		Rule:             r,
		RulePos:          0,
		originatingIndex: index,
		cause:            PREDICT_STATE,
		ctx:              ctx,
		from:             from,
	}
}

func newScanItem(sourceState *EarleyItem, index int, from []ItemId, ctx *completionContext) *EarleyItem {
	return &EarleyItem{
		Rule:                    sourceState.Rule,
		RulePos:                 sourceState.RulePos + 1,
		originatingIndex:        index,
		ctx:                     ctx,
		terminalSymbolsConsumed: sourceState.terminalSymbolsConsumed + 1,
		cause:                   SCAN_STATE,
		from:                    from,
	}
}

func newCompleteItem(sourceState *EarleyItem, from []ItemId, ctx *completionContext) *EarleyItem {
	return &EarleyItem{
		Rule:                    sourceState.Rule,
		RulePos:                 sourceState.RulePos + 1,
		originatingIndex:        sourceState.originatingIndex,
		terminalSymbolsConsumed: sourceState.terminalSymbolsConsumed,
		ctx:                     ctx,
		cause:                   COMPLETE_STATE,
		from:                    from,
	}
}

// I like this bit from gearley, so I am leaving it the way it was
func (item *EarleyItem) String() string {
	rightStrings := make([]string, len(item.Rule.right))
	for i, r := range item.Rule.right {
		rightStrings[i] = r.String()
	}

	fromStr := "root"
	if item.from != nil {
		fromStrings := make([]string, len(item.from))
		for i, r := range item.from {
			fromStrings[i] = fmt.Sprintf("S[%d][%d]", r.StateSetIndex, r.ItemIndex)
		}
		fromStr = strings.Join(fromStrings, ",")
	}
	return fmt.Sprintf("Rule(%v) -> %v %v %v (%d) (cause:%v) (tokensConsumed:%v) (from: %v)\n",
		item.Rule.left.String(),
		strings.Join(rightStrings[0:item.RulePos], " "),
		Cursor,
		strings.Join(rightStrings[item.RulePos:], " "),
		item.originatingIndex,
		item.cause,
		item.terminalSymbolsConsumed,
		fromStr,
	)
}

// i apologize but these are actually some practical limits though.
func (item *EarleyItem) badhash() uint64 {
	// let's just assume we don't have more than 1k rules,
	// or rules which are over 500 chars long,
	// or more than 500 symbols
	return uint64(item.Rule.grammarRuleId)<<32 | uint64(item.RulePos)<<16 | uint64(item.originatingIndex)
}

// complete means that dot reaches the end
func (item *EarleyItem) isCompleted() bool {
	return item.RulePos == item.Rule.length()
}

// check if is terminal and if is matching
func (item *EarleyItem) DoesTokenTypeMatch(tkn Tokhan) bool {
	s := item.GetRightSymbolByIndex(item.RulePos)
	return s.isMatchingTerminal(tkn.Type)
}

// check if is terminal and if is matching
func (item *EarleyItem) GetRightSymbolTypeByRulePos() *TokenType {
	s := item.GetRightSymbolByIndex(item.RulePos)
	return s.getType()
}

func (item *EarleyItem) GetRightSymbolByIndex(i int) Symbol {
	if i < len(item.Rule.right) {
		return item.Rule.right[i]
	}
	return Eof
}

// get the next symbol after the dot
func (item *EarleyItem) GetRightSymbolByRulePos() Symbol {
	return item.GetRightSymbolByIndex(item.RulePos)
}
