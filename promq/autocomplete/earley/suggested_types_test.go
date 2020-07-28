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
	"sort"
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
			expectedTypesList: [][]TokenType{{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN}},
		},
		{
			name:              "If we have an empty string, then we should suggest",
			inputString:       "",
			tokenPosList:      []int{0},
			expectedTypesList: [][]TokenType{{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN}},
		},
		{
			name:         "Binary Expression - scalar binary with arithmetic operation",
			inputString:  "123 + 4",
			tokenPosList: []int{1, 2, 3},
			expectedTypesList: [][]TokenType{
				{ARITHMETIC, COMPARISION, EOF},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{EOF, ARITHMETIC, COMPARISION},
			},
		},
		{
			name:         "Binary Expression - scalar binary with comparision operation",
			inputString:  "123 + 4 <= bool 10",
			tokenPosList: []int{1, 2, 3, 4, 5},
			expectedTypesList: [][]TokenType{
				{ARITHMETIC, COMPARISION, EOF},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{EOF, ARITHMETIC, COMPARISION},
				{BOOL_KW, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, METRIC_ID, LEFT_PAREN},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
			},
		},
		{
			name:              "Binary Expression - no suggestion because set operators only apply between two vectors",
			inputString:       "123 and 3",
			tokenPosList:      []int{2},
			expectedTypesList: [][]TokenType{{}},
		},
		{
			name:         "Binary Expression - vector binary with set operation",
			inputString:  "foo and bar",
			tokenPosList: []int{1, 2, 3},
			expectedTypesList: [][]TokenType{
				{ARITHMETIC, COMPARISION, SET, OFFSET_KW, LEFT_BRACKET, LEFT_BRACE, LEFT_PAREN, EOF},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, GROUP_KW, LEFT_PAREN},
				{OFFSET_KW, LEFT_PAREN, LEFT_BRACE, LEFT_BRACKET, SET, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:         "Binary Expression - one_to_one vector match with arithmetic operator",
			inputString:  "foo * on(test,) bar",
			tokenPosList: []int{2, 3, 4, 5, 6, 7, 8},
			expectedTypesList: [][]TokenType{
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, GROUP_KW, LEFT_PAREN},
				{LEFT_PAREN},
				{RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				{COMMA, RIGHT_PAREN},
				{RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				{GROUP_SIDE, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{SET, OFFSET_KW, LEFT_PAREN, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:         "Binary Expression - one_to_one vector match with set operator",
			inputString:  "foo and on(test,) bar",
			tokenPosList: []int{7, 8},
			expectedTypesList: [][]TokenType{
				{NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{SET, OFFSET_KW, LEFT_PAREN, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:         "Binary Expression - one_to_many vector match",
			inputString:  "foo / on(test,blub) group_left (bar,) bar",
			tokenPosList: []int{8, 9, 10, 11, 12, 13, 14},
			expectedTypesList: [][]TokenType{
				{GROUP_SIDE, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{LEFT_PAREN, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP},
				{METRIC_LABEL_SUBTYPE, RIGHT_PAREN, NUM, METRIC_ID, LEFT_PAREN, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP},
				{COMMA, RIGHT_PAREN, OFFSET_KW, LEFT_PAREN, COMPARISION, ARITHMETIC, LEFT_BRACE, SET},
				{METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
				{NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{SET, OFFSET_KW, LEFT_PAREN, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:         "Metric Expression - with labels",
			inputString:  "metric_name{label1='foo', label2='bar'}",
			tokenPosList: []int{1, 2, 3, 4, 5, 6, 10},
			expectedTypesList: [][]TokenType{
				{EOF, LEFT_BRACE, OFFSET_KW, LEFT_BRACKET, LEFT_PAREN, COMPARISION, ARITHMETIC, SET},
				{METRIC_LABEL_SUBTYPE},
				{LABELMATCH},
				{STRING},
				{RIGHT_BRACE, COMMA},
				{METRIC_LABEL_SUBTYPE},
				{EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:         "Metric Expression - metric name contains `:`",
			inputString:  "metric:name{label1='foo', label2='bar'}",
			tokenPosList: []int{1, 2, 3, 4, 5, 6, 10},
			expectedTypesList: [][]TokenType{
				{EOF, LEFT_BRACE, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				{METRIC_LABEL_SUBTYPE},
				{LABELMATCH},
				{STRING},
				{RIGHT_BRACE, COMMA},
				{METRIC_LABEL_SUBTYPE},
				{EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:              "Metric Expression - with offset",
			inputString:       "metric_name offset 5m",
			tokenPosList:      []int{2, 3},
			expectedTypesList: [][]TokenType{{DURATION}, {EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET}},
		},
		{
			name:              "Metric Expression - range vector selector",
			inputString:       "metric_name[3m] offset 5m",
			tokenPosList:      []int{2, 3, 4},
			expectedTypesList: [][]TokenType{{DURATION}, {RIGHT_BRACKET, COLON}, {EOF, OFFSET_KW}},
		},
		{
			name:         "Aggregation expression - the clause is after expression",
			inputString:  "sum(metric_name)",
			tokenPosList: []int{1, 2, 3, 4},
			expectedTypesList: [][]TokenType{
				{AGGR_KW, LEFT_PAREN},
				{METRIC_ID, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, LEFT_PAREN, COMPARISION, ARITHMETIC, SET},
				{AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:         "Aggregation expression - the clause is before expression",
			inputString:  "sum by (label1, labels) (metricname)",
			tokenPosList: []int{1, 2, 3, 4, 5, 7, 8},
			expectedTypesList: [][]TokenType{
				{AGGR_KW, LEFT_PAREN},
				{LEFT_PAREN},
				{RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				{RIGHT_PAREN, COMMA},
				{METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
				{LEFT_PAREN},
				{METRIC_ID, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
			},
		},
		{
			name:         "Aggregation expression - multiple label matchers",
			inputString:  "sum(metricname{label1='foo', label2='bar'})",
			tokenPosList: []int{4, 5, 6, 7, 8, 12, 13},
			expectedTypesList: [][]TokenType{
				{METRIC_LABEL_SUBTYPE},
				{LABELMATCH},
				{STRING},
				{RIGHT_BRACE, COMMA},
				{METRIC_LABEL_SUBTYPE},
				{RIGHT_PAREN, OFFSET_KW, ARITHMETIC, COMPARISION, SET},
				{AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:         "Aggregation expression - has label list",
			inputString:  "sum(metricname{label1='foo', label2='bar'}) by (label1, label2)",
			tokenPosList: []int{14, 15, 16, 17},
			expectedTypesList: [][]TokenType{
				{LEFT_PAREN},
				{RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				{RIGHT_PAREN, COMMA},
				{METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
			},
		},
		{
			name:         "Function expression - scalar function",
			inputString:  "scalar(metricname)",
			tokenPosList: []int{2, 3, 4},
			expectedTypesList: [][]TokenType{
				{RIGHT_PAREN, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				{OFFSET_KW, LEFT_PAREN, RIGHT_PAREN, LEFT_BRACE, COMPARISION, ARITHMETIC, SET},
				{EOF, ARITHMETIC, COMPARISION},
			},
		},
		{
			name:         "Function expression - have expression as arg",
			inputString:  "floor(metricname{foo!='bar'})",
			tokenPosList: []int{8, 9},
			expectedTypesList: [][]TokenType{
				{RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				{EOF, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:         "Function expression - have aggregation expression as arg",
			inputString:  "vector(sum(metricname{foo!='bar'}))",
			tokenPosList: []int{10, 11},
			expectedTypesList: [][]TokenType{
				{RIGHT_PAREN, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
				{RIGHT_PAREN, COMMA, AGGR_KW, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:         "Function expression - have multiple args",
			inputString:  "round(metricname, 5)",
			tokenPosList: []int{3, 4, 5},
			expectedTypesList: [][]TokenType{
				{RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, LEFT_PAREN, LEFT_BRACE, COMPARISION, ARITHMETIC, SET},
				{METRIC_ID, NUM, AGGR_OP, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, LEFT_PAREN},
				{RIGHT_PAREN, COMMA, COMPARISION, ARITHMETIC},
			},
		},
		{
			name:         "Function expression - nested function call",
			inputString:  "ceil(abs(metricname{foo!='bar'}))",
			tokenPosList: []int{3, 4, 10, 11},
			expectedTypesList: [][]TokenType{
				{LEFT_PAREN},
				{RIGHT_PAREN, METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				{RIGHT_PAREN, COMMA, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:         "Subquery expression - expr is vectorSelector",
			inputString:  "metricname{foo='bar'}[10m:6s]",
			tokenPosList: []int{6, 7, 8, 9, 10, 11},
			expectedTypesList: [][]TokenType{
				{EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				{DURATION},
				{COLON, RIGHT_BRACKET},
				{DURATION, RIGHT_BRACKET},
				{RIGHT_BRACKET},
				{EOF, OFFSET_KW},
			},
		},
		{
			name:         "Subquery expression - expr is function expression",
			inputString:  "rate(metricname{foo='bar'}[5m])[10m:6s]",
			tokenPosList: []int{12, 13, 14, 15, 16, 17},
			expectedTypesList: [][]TokenType{
				{EOF, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				{DURATION},
				{COLON},
				{DURATION, RIGHT_BRACKET},
				{RIGHT_BRACKET},
				{EOF, OFFSET_KW},
			},
		},
		{
			name:         "Parentheses expression - number arithmetic",
			inputString:  "1 + 2/(3*1)",
			tokenPosList: []int{4, 5, 6, 7, 8, 9},
			expectedTypesList: [][]TokenType{
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{COMPARISION, ARITHMETIC},
				{NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{RIGHT_PAREN, ARITHMETIC, COMPARISION},
				{COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:         "Parentheses expression - matrix type",
			inputString:  "(foo + bar{nm='val'})[5m:] offset 10m",
			tokenPosList: []int{9, 10, 11, 12, 13, 14, 15, 16},
			expectedTypesList: [][]TokenType{
				{RIGHT_PAREN, OFFSET_KW, ARITHMETIC, COMPARISION, SET},
				{LEFT_BRACKET, ARITHMETIC, COMPARISION, SET, EOF},
				{DURATION},
				{COLON},
				{RIGHT_BRACKET, DURATION},
				{EOF, OFFSET_KW},
				{DURATION},
				{EOF},
			},
		},
		{
			name:         "Parentheses expression - nested parentheses",
			inputString:  "((foo + bar{nm='val'}) + metric_name) + 1",
			tokenPosList: []int{0, 1, 11, 12, 13, 14, 15, 16},
			expectedTypesList: [][]TokenType{
				{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				{RIGHT_PAREN, SET, ARITHMETIC, COMPARISION},
				{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, GROUP_KW},
				{OFFSET_KW, LEFT_BRACE, LEFT_PAREN, COMPARISION, SET, ARITHMETIC, RIGHT_PAREN},
				{EOF, LEFT_BRACKET, COMPARISION, SET, ARITHMETIC},
				{METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, GROUP_KW},
				{EOF, COMPARISION, SET, ARITHMETIC, LEFT_BRACKET},
			},
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

				if !isEqualTypes(tknTypes, tc.expectedTypesList[i]) {
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
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, LEFT_PAREN, SET},
		},
		{
			"new input is empty",
			"sum(metric_name_one",
			"",
			[]TokenType{METRIC_ID, NUM, AGGR_OP, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, LEFT_PAREN},
		},
		{
			"previous input is empty",
			"",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, LEFT_PAREN, SET},
		},
		{
			"previous input and new input are different from beginning",
			"metric_name{label=",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, LEFT_PAREN, SET},
		},
		{
			"previous input and new input are partially same",
			"sum(metric_name_one{",
			"sum(metric_name_one)",
			[]TokenType{AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
		},
		{
			"new input covers previous input",
			"sum(metric_name_one",
			"sum(metric_name_one)",
			[]TokenType{AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
		},
		{
			"previous input covers new input",
			"sum(metric_name_one{",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, LEFT_PAREN, SET},
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
			if !isEqualTypes(tknTypes, tc.expectedTypes) {
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

func isEqualTypes(actual []TokenType, expected []TokenType) bool {
	if len(actual) == 0 && len(expected) == 0 {
		return true
	}
	sort.Slice(actual, func(i, j int) bool {
		return actual[i] > actual[j]
	})
	sort.Slice(expected, func(k, j int) bool {
		return expected[k] > expected[j]
	})
	if !reflect.DeepEqual(actual, expected) {
		return false
	}
	return true
}
