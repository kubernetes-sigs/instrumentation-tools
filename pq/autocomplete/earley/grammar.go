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

type GrammarRule struct {
	left          NonTerminalNode
	right         []Symbol
	grammarRuleId int
}

func (r *GrammarRule) length() int {
	return len(r.right)
}
func (r *GrammarRule) String() string {
	rightStrings := make([]string, len(r.right))
	for i, s := range r.right {
		rightStrings[i] = s.String()
	}
	return fmt.Sprintf("%v -> %v", r.left.String(), strings.Join(rightStrings, " "))
}

func NewRule(t NonTerminalNode, symbols ...Symbol) *GrammarRule {
	return &GrammarRule{
		left:  t,
		right: symbols,
	}
}

type Grammar struct {
	rules []*GrammarRule
}

func NewGrammar(rules ...*GrammarRule) *Grammar {
	// let's add the index that the rule was added to the grammar to use as an RuleId.
	for i, r := range rules {
		r.grammarRuleId = i
	}
	return &Grammar{rules: rules}
}

// initial earley state sets contain all rules
func (g Grammar) initialStateSet() *StateSet {
	ss := NewStateSet()
	for _, r := range g.rules {
		if r.left.isRoot() {
			item := &EarleyItem{Rule: r, RulePos: 0, originatingIndex: 0, cause: "predict"}
			ss.Add(item)
		}

	}
	return ss
}

// return the rules that left is input Symbol
func (g *Grammar) recognizedRules(s Symbol) (found []*GrammarRule) {
	for _, r := range g.rules {
		if r.left == s {
			found = append(found, r)
		}
	}
	return found
}

// Earley parsers parse symbols in a bottom up manner
type Symbol interface {
	isTerminal() bool
	String() string
	getType() *TokenType
	isMatchingTerminal(tokhanType TokenType) bool
	// s and input are slices of the full state set and the full input.
}

type EarleyNode interface {
	isTerminal() bool
	String() string
	getType() *TokenType
	isMatchingTerminal(tokhanType TokenType) bool
}

type nonTerminal struct {
	name string
	root bool
}

type NonTerminalNode interface {
	EarleyNode
	GetName() string
	isRoot() bool
}

func (nt nonTerminal) GetName() string {
	return nt.name
}

func NewNonTerminal(name string, root bool) NonTerminalNode {
	return nonTerminal{name: name, root: root}
}

func (nt nonTerminal) isTerminal() bool {
	return false
}
func (nt nonTerminal) isRoot() bool {
	return nt.root
}
func (nt nonTerminal) String() string {
	return nt.name
}

func (nt nonTerminal) getType() *TokenType {
	return nil
}

func (nt nonTerminal) isMatchingTerminal(TokenType) bool {
	return false
}

type terminal struct {
	tokenType    TokenType
	tokenSubType *TokenType
}

func NewTerminal(name TokenType) EarleyNode {
	return terminal{tokenType: name}
}

func NewTerminalWithSubType(name TokenType, subtype TokenType) EarleyNode {
	return terminal{tokenType: name, tokenSubType: &subtype}
}

func (t terminal) isTerminal() bool {
	return true
}

func (t terminal) String() string {
	if t.tokenSubType != nil {
		return fmt.Sprintf("'%v'", *t.tokenSubType)
	}
	return fmt.Sprintf("'%v'", t.tokenType)
}

func (t terminal) isMatchingTerminal(tt TokenType) bool {
	return t.tokenType == tt || (t.tokenSubType != nil && *t.tokenSubType == tt)
}
func (t terminal) getType() *TokenType {
	if t.tokenSubType != nil {
		return t.tokenSubType
	}
	return &t.tokenType
}
