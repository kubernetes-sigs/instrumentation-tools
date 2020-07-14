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

// luthor is our lexer, of course.

import (
	"fmt"
	"strings"

	"k8s.io/instrumentation-tools/debug"

	"github.com/prometheus/prometheus/promql/parser"
)

type Tokens []Tokhan

func (ws Tokens) Vals() []string {
	v := make([]string, len(ws))
	for i, w := range ws {
		v[i] = w.Val
	}
	return v
}

func (ws Tokens) Types() []string {
	v := make([]string, len(ws))
	for i, w := range ws {
		v[i] = string(w.Type)
	}
	return v
}
func (ws Tokens) Print() {
	for _, w := range ws {
		debug.Debugln(w.String())
	}
}
func (ws Tokens) PrintVals() {
	b := strings.Builder{}

	for _, w := range ws {
		b.WriteString(w.Val)
		b.WriteString("|")
	}
	fmt.Println(b.String())
}

func (ws Tokens) Last() Tokhan {
	return ws[len(ws)-2]
}

type TypedToken interface {
	GetTokenType() TokenType
}

type TokenType string

const (
	ID                   TokenType = "identifier"
	METRIC_ID            TokenType = "metric-identifier"
	METRIC_LABEL_SUBTYPE TokenType = "metric-label-identifier"
	ARITHMETIC           TokenType = "arithmetic"
	OPERATOR             TokenType = "operator"
	AGGR_OP              TokenType = "aggregator_operation"
	AGGR_KW              TokenType = "aggregator_keyword"
	LEFT_BRACE           TokenType = "leftbrace"
	RIGHT_BRACE          TokenType = "rightbrace"
	LEFT_PAREN           TokenType = "leftparen"
	COMMA                TokenType = "comma"
	RIGHT_PAREN          TokenType = "rightparen"
	STRING               TokenType = "string"
	NUM                  TokenType = "number"
	EOF                  TokenType = "EOF"
	UNKNOWN              TokenType = "unknown"
)

// Tokhan contains the essential bits of data we need
// for processing a single lexical unit.
type Tokhan struct {
	StartPos int
	EndPos   int
	Type     TokenType
	ItemType parser.ItemType
	Val      string
	_index   int
}

func (t Tokhan) isEof() bool {
	return t.ItemType == parser.EOF
}

func (t Tokhan) String() string {
	return fmt.Sprintf("Tokhan.Val(%v) Type(%v) StartEnd[%v:%v]",
		t.Val,
		t.Type,
		t.StartPos,
		t.EndPos,
	)
}

func extractWords(query string) Tokens {
	words := extractTokensWithOffset(query, 0)
	words.Print()
	return words
}

// todo(rant):  we should probably just hand-roll a parser for this. I am
// todo(cont):  not fond of the way this lexer encodes random syntactical
// todo(cont):  rules during lexing
func extractTokensWithOffset(query string, offset int) (words Tokens) {
	l := parser.Lex(query)
	i := 0
	for {
		currItem := parser.Item{}
		l.NextItem(&currItem)
		if currItem.Typ == parser.EOF {
			words = append(words, createTokenFromItem(currItem, offset))
			break
		}

		// recurse our lexer on a sub-query string. We do this specifically to accommodate
		// strings like `start(label='value)end` where we want as output:
		// a linked list of tokens like this:
		// "start" <-> "(" <-> "label" <-> "=" <-> "'value" <-> ")" <-> "end"
		if currItem.Typ == parser.ERROR {
			substring := query[currItem.Pos:]
			// we're recursing and found an error already abort
			if i == 0 {
				break
			}
			subWords := extractTokensWithOffset(substring, int(currItem.Pos))
			if len(subWords) > 0 {
				words = append(words, subWords...)
			}
			return
		}
		words = append(words, createTokenFromItem(currItem, offset))
		i++
	}
	return
}

func createTokenFromItem(item parser.Item, offset int) Tokhan {
	return Tokhan{
		Val:      item.Val,
		ItemType: item.Typ,
		Type:     mapParserItemTypeToTokhanType(item),
		StartPos: int(item.Pos) + offset,
		EndPos:   int(item.PositionRange().End),
	}
}

func mapParserItemTypeToTokhanType(item parser.Item) TokenType {
	t := item.Typ
	switch {
	case item.Val == "by", item.Val == "without":
		return AGGR_KW
	case t == parser.EOF:
		return EOF
	case t == parser.STRING:
		return STRING
	case t.IsAggregator():
		return AGGR_OP
	case t == parser.IDENTIFIER, t == parser.METRIC_IDENTIFIER:
		return ID
	case t == parser.LEFT_BRACE:
		return LEFT_BRACE
	case t == parser.RIGHT_BRACE:
		return RIGHT_BRACE
	case t == parser.LEFT_PAREN:
		return LEFT_PAREN
	case t == parser.RIGHT_PAREN:
		return RIGHT_PAREN
	case t == parser.ADD, t == parser.SUB, t == parser.MUL, t == parser.DIV:
		return ARITHMETIC
	case t == parser.COMMA:
		return COMMA
	case t == parser.EQL:
		return OPERATOR
	case t == parser.NUMBER:
		return NUM
	default:
		return UNKNOWN
	}
}
