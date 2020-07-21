package earley

import (
	"fmt"

	"sigs.k8s.io/instrumentation-tools/debug"
)

type StateType string

const (
	PREDICT_STATE  StateType = "predict"
	SCAN_STATE     StateType = "scan"
	COMPLETE_STATE StateType = "complete"
)

// EarleyStateSets are specific to the position which we have
// so far parsed. For position 0, this means all rules from
// our grammar which start with a non-terminal are possible. So,
// an EarleyStateSet in an earleyChart at position 0 will contain
// all the rules in our grammar which start with a non-terminal symbol.
// Every time we consume a symbol, we look back at our chart from N-1
// we advance the symbol pointer + 1, or we shed the Earley item if it
// turns out we cannot actually use that rule.

// An Earley StateSet represents all possible rules at a given index in an Earley chart.
type StateSet struct {
	stateNo int
	items   []*EarleyItem
	itemSet map[uint64]bool
}

func NewStateSet() *StateSet {
	return &StateSet{
		itemSet: make(map[uint64]bool),
	}
}

func (s *StateSet) String() string {
	return fmt.Sprint(s.GetStates())
}
func (s *StateSet) GetStates() (states []EarleyItem) {
	for _, i := range s.items {
		if i != nil {
			states = append(states, *i)
		} else {
			debug.Debugf("WTF we had a nil EarleyItem in our list")
		}
	}
	return states
}
func (s *StateSet) Length() int {
	return len(s.itemSet)
}

// idempotent put operation
func (s *StateSet) Add(item *EarleyItem) bool {
	if _, ok := s.itemSet[item.badhash()]; ok {
		return false
	}
	s.itemSet[item.badhash()] = true
	item.id.StateSetIndex = s.stateNo
	item.id.ItemIndex = len(s.items)
	s.items = append(s.items, item)
	return true
}
func (s *StateSet) findItemsToComplete(symbol NonTerminalNode) (candidates []EarleyItem) {
	for _, item := range s.GetStates() {

		if item.isCompleted() {
			continue
		}

		// Only non-terminals can be added to stateset in completion operation
		switch c := item.Rule.right[item.RulePos].(type) {
		case nonTerminal:
			if c.name == symbol.GetName() {
				candidates = append(candidates, item)
			}
		default:
			// continue since we can't complete a terminal
		}
	}
	return candidates
}
