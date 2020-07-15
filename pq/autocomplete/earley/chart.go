package earley

import (
	"fmt"
	"strings"

	"k8s.io/instrumentation-tools/debug"
)

// Earley parsers produce earley charts as an output, which
// is nested data structure.

// Earley charts represent the accumulated state during our
// parse. Given a full earley parse, we will end up with an
// earley chart with N+1 items, where N is the number of symbols
// consumed in the parse. We end up with N+1 because we start an
// earley parse by having consumed zero symbols. For each symbol,
// we consume during our earley parse, we have an earley state set,
// which consists of earley items.

// EarleyStateSets are specific to the position which we have
// so far parsed. For position 0, this means all rules from
// our grammar which start with a non-terminal are possible. So,
// an EarleyStateSet in an earleyChart at position 0 will contain
// all the rules in our grammar which start with a non-terminal symbol.
// Every time we consume a symbol, we look back at our chart from N-1
// we advance the symbol pointer + 1, or we shed the Earley item if it
// turns out we cannot actually use that rule.

// Earley items are simple data structures meant to contain the following
// bits:
//      (1) a rule in the grammar which has so far been valid
//      (2) the position in the rule that denotes how much of that rule
//          has been consumed by our parser
//      (3) a reference to the index in our symbol list that we've consumed
//          (this probably isn't fully necessary, I could imagine that we could
//           restructure the earley parser such that it takes a earleyStateSet
//           representing the past state, and construct the next stateSet with an arbitrary
//           symbol passed in).
//noinspection GoNameStartsWithPackageName
type EarleyChart interface {
	States() []*StateSet
	GetState(insertionOrderZeroIndexed int) *StateSet
	String() string
}

type earleyChart struct {
	inputWords Tokens
	state      []*StateSet
}

func initializeChart(g Grammar) *earleyChart {
	initialSet := g.initialStateSet()
	initialSet.stateNo = 0
	initialSets := []*StateSet{
		initialSet,
	}
	return &earleyChart{nil, initialSets}
}

func (c *earleyChart) Length() int {
	return len(c.state)
}
func (c *earleyChart) States() []*StateSet {
	return c.state
}
func (c *earleyChart) GetState(insertionOrderZeroIndexed int) *StateSet {
	// validate boundary conditions
	if insertionOrderZeroIndexed < len(c.state) {
		return c.state[insertionOrderZeroIndexed]
	}
	return nil
}

func (c *earleyChart) setInputWords(t Tokens) {
	c.inputWords = t
}

func (c *earleyChart) GetValidTerminalTypesAtStateSet(wordIndex int) (types []ContextualToken) {
	// todo(han): ensure that we have actually parsed up to wordIndex
	terminalTypes := map[TokenType]bool{}
	state := c.GetState(wordIndex)
	for _, item := range state.items {
		rhs := item.Rule.right
		// finished already, this isn't valid for completion
		if item.terminalSymbolsConsumed == wordIndex+1 {
			continue
		}
		// boundary error, can't be an applicable rule
		if len(rhs) <= item.RulePos {
			debug.Debugf("ABORT %v %v\n", wordIndex, rhs)
			continue
		}

		switch sym := rhs[item.RulePos].(type) {
		case terminal:
			tkn := sym.tokenType
			if sym.tokenSubType != nil {
				tkn = *sym.tokenSubType
			}
			tknCtx := &ContextualToken{TokenType: tkn}
			if _, ok := terminalTypes[tkn]; !ok {
				terminalTypes[tkn] = true
				if tkn != METRIC_ID {
					tknCtx.ctx = item.ctx
				}
				types = append(types, *tknCtx)
			}
		default:
			// continue since we can't complete a terminal
		}
	}
	return
}

// this one gets printed out everywhere, because we need to look at these.
func (c *earleyChart) String() string {
	sb := strings.Builder{}
	sb.WriteString("earleyChart.String()\n")
	sb.WriteString(fmt.Sprint("Input word: ", strings.Join(c.inputWords.Vals(), ", "), "\n"))

	inputTokenTypes := c.inputWords.Types()
	for i, s := range c.state {
		currentInput := fmt.Sprintf("%v %v %v", strings.Join(inputTokenTypes[0:i], " "), Cursor, strings.Join(inputTokenTypes[i:], " "))
		sb.WriteString(fmt.Sprint("State", " ", i, " ", currentInput, "\n"))
		for j, item := range s.items {
			sb.WriteString(fmt.Sprint(j, " ", item.String()))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

func (c *earleyChart) append(set *StateSet) {
	set.stateNo = len(c.state)
	c.state = append(c.state, set)
}
func (c *earleyChart) putAt(index int, set *StateSet) {
	set.stateNo = index
	c.state[index] = set
}

// todo: consider downsizing the chart size?
func (c *earleyChart) invalidateStartingAt(index int) {
	for i := index; i < c.Length(); i++ {
		c.putAt(i, NewStateSet())
	}
}
