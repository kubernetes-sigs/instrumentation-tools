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
	"reflect"
	"testing"
)

func TestEarleyItems(t *testing.T) {
	testCases := []struct {
		desc                string
		rule                *GrammarRule
		rulePos             int
		expectedIsCompleted bool
	}{
		{
			desc:                "shouldn't be completed",
			rule:                NewRule(LabelValueExpression, LBrace, Identifier, Operator, Str, RBrace, Eof),
			rulePos:             0,
			expectedIsCompleted: false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			item := &EarleyItem{
				Rule:    tc.rule,
				RulePos: 5,
			}

			if item.isCompleted() {
				t.Errorf("Rule shouldn't be completed")
			}
		})
	}
}

func TestCompletionContext(t *testing.T) {
	testCases := []struct {
		name           string
		inputString    string
		expectedMetric *string
		expectedLabels *string
	}{
		//	Todo(yuchen): add test cases
	}

	for _, tc := range testCases {
		p := NewEarleyParser(*promQLGrammar)
		tokens := extractWords(tc.inputString)
		validTypes := p.GetSuggestedTokenType(tokens)

		var metricName *string
		for _, ct := range validTypes {
			if ct.ctx != nil {
				if ct.ctx.metric != nil {
					metricName = ct.ctx.metric
				}
			}
		}
		t.Errorf("Got %v, Expected metric in context :%v ",
			safeRead(metricName),
			safeRead(tc.expectedMetric))
	}
}

func TestSuggestedTypes(t *testing.T) {
	testCases := []struct {
		name              string
		inputString       string
		tokenPosList      []int
		expectedTypesList [][]TokenType
	}{
		{
			name:              "If we've consumed zero tokens, then we should suggest",
			inputString:       "blah",
			tokenPosList:      []int{0},
			expectedTypesList: [][]TokenType{{METRIC_ID, NUM, AGGR_OP}},
		},
		{
			name:              "If we have an empty string, then we should suggest",
			inputString:       "",
			tokenPosList:      []int{0},
			expectedTypesList: [][]TokenType{{METRIC_ID, NUM, AGGR_OP}},
		},
		{
			name:              "Binary Expression - with scalar",
			inputString:       "123 + 4 == bool 10",
			tokenPosList:      []int{0, 1, 2, 3, 4, 5},
			expectedTypesList: [][]TokenType{{METRIC_ID, NUM, AGGR_OP}, {ARITHMETIC, COMPARISION}, {NUM}, {EOF, ARITHMETIC, COMPARISION}, {BOOL_KW}, {NUM}},
		},
		{
			name:              "Metric Expression",
			inputString:       "metric_name{label1='foo', label2='bar'}",
			tokenPosList:      []int{1, 2, 3, 4, 5},
			expectedTypesList: [][]TokenType{{EOF, LEFT_BRACE}, {METRIC_LABEL_SUBTYPE}, {LABELMATCH}, {STRING}, {RIGHT_BRACE, COMMA}, {METRIC_LABEL_SUBTYPE}},
		},
		{
			name:              "Aggregation expression - the clause is after expression",
			inputString:       "sum(metric_name)",
			tokenPosList:      []int{1, 2, 3, 4},
			expectedTypesList: [][]TokenType{{AGGR_KW, LEFT_PAREN}, {METRIC_ID}, {RIGHT_PAREN, LEFT_BRACE}, {AGGR_KW, EOF}},
		},
		{
			name:              "Aggregation expression - the clause is before expression",
			inputString:       "sum by (label1, labels) (metricname)",
			tokenPosList:      []int{1, 2, 3, 4, 5, 7, 8},
			expectedTypesList: [][]TokenType{{AGGR_KW, LEFT_PAREN}, {LEFT_PAREN}, {RIGHT_PAREN, METRIC_LABEL_SUBTYPE}, {RIGHT_PAREN, COMMA}, {METRIC_LABEL_SUBTYPE}, {LEFT_PAREN}, {METRIC_ID}},
		},
		{
			name:              "Aggregation expression - multiple label matchers",
			inputString:       "sum(metricname{label1='foo', label2='bar'})",
			tokenPosList:      []int{4, 5, 6, 7, 8, 12, 13},
			expectedTypesList: [][]TokenType{{METRIC_LABEL_SUBTYPE}, {LABELMATCH}, {STRING}, {RIGHT_BRACE, COMMA}, {METRIC_LABEL_SUBTYPE}, {RIGHT_PAREN}, {AGGR_KW, EOF}},
		},
		{
			name:              "Aggregation expression - has label list",
			inputString:       "sum(metricname{label1='foo', label2='bar'}) by (label1, label2)",
			tokenPosList:      []int{14, 15, 16, 17},
			expectedTypesList: [][]TokenType{{LEFT_PAREN}, {RIGHT_PAREN, METRIC_LABEL_SUBTYPE}, {RIGHT_PAREN, COMMA}, {METRIC_LABEL_SUBTYPE}},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewEarleyParser(*promQLGrammar)
			chart := p.Parse(tc.inputString)
			for i, pos := range tc.tokenPosList {
				validTypes := chart.GetValidTerminalTypesAtStateSet(pos)
				var tknTypes []TokenType
				for _, ct := range validTypes {
					tknTypes = append(tknTypes, ct.TokenType)
				}
				if !reflect.DeepEqual(tknTypes, tc.expectedTypesList[i]) {
					t.Errorf("Position %d: Got %v, expected %v\n", pos, tknTypes, tc.expectedTypesList[i])
				}
			}

		})
	}
}

func TestPartialParse(t *testing.T) {
	testCases := []struct {
		name          string
		prevInput     string
		newInput      string
		expectedTypes []TokenType
	}{
		{
			"new input is same as previous input",
			"sum(metric_name_one",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE},
		},
		{
			"new input is empty",
			"sum(metric_name_one",
			"",
			[]TokenType{METRIC_ID, NUM, AGGR_OP},
		},
		{
			"previous input is empty",
			"",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE},
		},
		{
			"previous input and new input are different from beginning",
			"metric_name{label=",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE},
		},
		{
			"previous input and new input are partially same",
			"sum(metric_name_one{",
			"sum(metric_name_one)",
			[]TokenType{AGGR_KW, EOF},
		},
		{
			"new input covers previous input",
			"sum(metric_name_one",
			"sum(metric_name_one)",
			[]TokenType{AGGR_KW, EOF},
		},
		{
			"previous input covers new input",
			"sum(metric_name_one{",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewEarleyParser(*promQLGrammar)
			p.Parse(tc.prevInput)
			tokens := extractWords(tc.newInput)
			validTypes := p.GetSuggestedTokenType(tokens)

			var tknTypes []TokenType
			for _, ct := range validTypes {
				tknTypes = append(tknTypes, ct.TokenType)
			}
			if !reflect.DeepEqual(tknTypes, tc.expectedTypes) {
				t.Errorf("\nGot %v, expected %v\n", validTypes, tc.expectedTypes)
			}
		})
	}
}

func safeRead(sp *string) string {
	if sp == nil {
		return ""
	}
	return *sp
}
