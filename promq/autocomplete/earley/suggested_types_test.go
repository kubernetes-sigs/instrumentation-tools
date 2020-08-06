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
		name                         string
		inputString                  string
		expectedTypesFromParsePosMap map[int][]TokenType
	}{
		{
			name:        "If we've consumed zero tokens, then we should suggest",
			inputString: "blah",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				0: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
			},
		},
		{
			name:        "If we have an empty string, then we should suggest",
			inputString: "",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				0: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
			},
		},
		{
			name:        "Binary Expression - scalar binary with arithmetic operation",
			inputString: "123 + 4",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {ARITHMETIC, COMPARISION, EOF},
				2: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				3: {EOF, ARITHMETIC, COMPARISION},
			},
		},
		{
			name:        "Binary Expression - with unary expression",
			inputString: "123 + (-4)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				3: {UNARY_OP, NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				4: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				5: {RIGHT_PAREN, COMPARISION, ARITHMETIC},
				6: {EOF, ARITHMETIC, COMPARISION},
			},
		},
		{
			name:        "Binary Expression - scalar binary with comparision operation",
			inputString: "123 + 4 <= bool 10",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {ARITHMETIC, COMPARISION, EOF},
				2: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				3: {EOF, ARITHMETIC, COMPARISION},
				4: {BOOL_KW, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, METRIC_ID, LEFT_PAREN},
				5: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
			},
		},
		{
			name:        "Binary Expression - no suggestion because set operators only apply between two vectors",
			inputString: "123 and 3",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {},
			},
		},
		{
			name:        "Binary Expression - vector binary with set operation",
			inputString: "foo and bar",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {ARITHMETIC, COMPARISION, SET, OFFSET_KW, LEFT_BRACKET, LEFT_BRACE, EOF},
				2: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, GROUP_KW, LEFT_PAREN},
				3: {OFFSET_KW, LEFT_BRACE, LEFT_BRACKET, SET, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:        "Binary Expression - one_to_one vector match with arithmetic operator",
			inputString: "foo * on(test,) bar",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, GROUP_KW, LEFT_PAREN},
				3: {LEFT_PAREN},
				4: {RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				5: {COMMA, RIGHT_PAREN},
				6: {RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				7: {GROUP_SIDE, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				8: {SET, OFFSET_KW, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:        "Binary Expression - one_to_one vector match with set operator",
			inputString: "foo and on(test,) bar",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				7: {NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				8: {SET, OFFSET_KW, LEFT_BRACE, LEFT_BRACKET, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:        "Binary Expression - one_to_many vector match",
			inputString: "foo / on(test,blub) group_left (bar,) bar",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				8:  {GROUP_SIDE, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				9:  {LEFT_PAREN, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP},
				10: {METRIC_LABEL_SUBTYPE, RIGHT_PAREN, NUM, METRIC_ID, LEFT_PAREN, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, UNARY_OP},
				11: {COMMA, RIGHT_PAREN, OFFSET_KW, COMPARISION, ARITHMETIC, LEFT_BRACE, SET},
				12: {METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
				13: {NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				14: {SET, OFFSET_KW, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:        "Metric Expression - with labels",
			inputString: "metric_name{label1='foo', label2='bar'}",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1:  {EOF, LEFT_BRACE, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				2:  {METRIC_LABEL_SUBTYPE},
				3:  {LABELMATCH},
				4:  {STRING},
				5:  {RIGHT_BRACE, COMMA},
				6:  {METRIC_LABEL_SUBTYPE},
				10: {EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:        "Metric Expression - metric name contains `:`",
			inputString: "metric:name{label1='foo', label2='bar'}",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1:  {EOF, LEFT_BRACE, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				2:  {METRIC_LABEL_SUBTYPE},
				3:  {LABELMATCH},
				4:  {STRING},
				5:  {RIGHT_BRACE, COMMA},
				6:  {METRIC_LABEL_SUBTYPE},
				10: {EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:        "Metric Expression - with offset",
			inputString: "metric_name offset 5m",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {DURATION},
				3: {EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:        "Metric Expression - range vector selector",
			inputString: "metric_name[3m] offset 5m",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {DURATION},
				3: {RIGHT_BRACKET, COLON},
				4: {EOF, OFFSET_KW},
			},
		},
		{
			name:        "Aggregation expression - only metric",
			inputString: "sum(metric_name)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {AGGR_KW, LEFT_PAREN},
				2: {METRIC_ID, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
				3: {RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
				4: {AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:        "Aggregation expression - the clause is before expression",
			inputString: "sum by (label1, labels) (metricname)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {AGGR_KW, LEFT_PAREN},
				2: {LEFT_PAREN},
				3: {RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				4: {RIGHT_PAREN, COMMA},
				5: {METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
				7: {LEFT_PAREN},
				8: {METRIC_ID, NUM, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN},
			},
		},
		{
			name:        "Aggregation expression - multiple label matchers",
			inputString: "sum(metricname{label1='foo', label2='bar'})",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				4:  {METRIC_LABEL_SUBTYPE},
				5:  {LABELMATCH},
				6:  {STRING},
				7:  {RIGHT_BRACE, COMMA},
				8:  {METRIC_LABEL_SUBTYPE},
				12: {RIGHT_PAREN, OFFSET_KW, ARITHMETIC, COMPARISION, SET},
				13: {AGGR_KW, EOF, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:        "Aggregation expression - the clause is after expression",
			inputString: "sum(metricname{label1='foo', label2='bar'}) by (label1, label2)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				14: {LEFT_PAREN},
				15: {RIGHT_PAREN, METRIC_LABEL_SUBTYPE},
				16: {RIGHT_PAREN, COMMA},
				17: {METRIC_LABEL_SUBTYPE, RIGHT_PAREN},
			},
		},
		{
			name:        "Function expression - scalar function",
			inputString: "scalar(metricname)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				2: {RIGHT_PAREN, NUM, METRIC_ID, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, AGGR_OP, LEFT_PAREN, UNARY_OP},
				3: {OFFSET_KW, RIGHT_PAREN, LEFT_BRACE, COMPARISION, ARITHMETIC, SET},
				4: {EOF, ARITHMETIC, COMPARISION},
			},
		},
		{
			name:        "Function expression - have expression as arg",
			inputString: "floor(metricname{foo!='bar'})",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				1: {LEFT_PAREN},
				8: {RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				9: {EOF, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:        "Function expression - have aggregation expression as arg",
			inputString: "vector(sum(metricname{foo!='bar'}))",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				10: {RIGHT_PAREN, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
				11: {RIGHT_PAREN, COMMA, AGGR_KW, COMPARISION, ARITHMETIC, LEFT_BRACKET, SET},
			},
		},
		{
			name:        "Function expression - have multiple args",
			inputString: "round(metricname, -5)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				3: {RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, LEFT_BRACE, COMPARISION, ARITHMETIC, SET},
				4: {METRIC_ID, NUM, AGGR_OP, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, LEFT_PAREN, UNARY_OP},
				5: {METRIC_ID, NUM, AGGR_OP, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, LEFT_PAREN},
				6: {RIGHT_PAREN, COMMA, COMPARISION, ARITHMETIC},
			},
		},
		{
			name:        "Function expression - nested function call",
			inputString: "ceil(abs(metricname{foo!='bar'}))",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				3:  {LEFT_PAREN},
				4:  {RIGHT_PAREN, METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				10: {RIGHT_PAREN, COMMA, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				11: {RIGHT_PAREN, COMMA, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
			},
		},
		{
			name:        "Subquery expression - expr is vectorSelector",
			inputString: "metricname{foo='bar'}[10m:6s]",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				6:  {EOF, OFFSET_KW, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				7:  {DURATION},
				8:  {COLON, RIGHT_BRACKET},
				9:  {DURATION, RIGHT_BRACKET},
				10: {RIGHT_BRACKET},
				11: {EOF, OFFSET_KW},
			},
		},
		{
			name:        "Subquery expression - expr is function expression",
			inputString: "rate(metricname{foo='bar'}[5m])[10m:6s]",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				12: {EOF, LEFT_BRACKET, COMPARISION, ARITHMETIC, SET},
				13: {DURATION},
				14: {COLON},
				15: {DURATION, RIGHT_BRACKET},
				16: {RIGHT_BRACKET},
				17: {EOF, OFFSET_KW},
			},
		},
		{
			name:        "Parentheses expression - number arithmetic",
			inputString: "1 + 2/(3*1)",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				4: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				5: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				6: {COMPARISION, ARITHMETIC},
				7: {NUM, METRIC_ID, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				8: {RIGHT_PAREN, ARITHMETIC, COMPARISION},
				9: {COMPARISION, ARITHMETIC, EOF},
			},
		},
		{
			name:        "Parentheses expression - matrix type",
			inputString: "(foo + bar{nm='val'})[5m:] offset 10m",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				9:  {RIGHT_PAREN, OFFSET_KW, ARITHMETIC, COMPARISION, SET},
				10: {LEFT_BRACKET, ARITHMETIC, COMPARISION, SET, EOF},
				11: {DURATION},
				12: {COLON},
				13: {RIGHT_BRACKET, DURATION},
				14: {EOF, OFFSET_KW},
				15: {DURATION},
				16: {EOF},
			},
		},
		{
			name:        "Parentheses expression - nested parentheses",
			inputString: "((foo + bar{nm='val'}) + metric_name) + 1",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				0:  {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				1:  {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				11: {RIGHT_PAREN, SET, ARITHMETIC, COMPARISION},
				12: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, GROUP_KW},
				13: {OFFSET_KW, LEFT_BRACE, COMPARISION, SET, ARITHMETIC, RIGHT_PAREN},
				14: {EOF, LEFT_BRACKET, COMPARISION, SET, ARITHMETIC},
				15: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, GROUP_KW},
				16: {EOF, COMPARISION, SET, ARITHMETIC, LEFT_BRACKET},
			},
		},
		{
			name:        "Unary expression - number",
			inputString: "-1 + 2 * 5",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				0: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				1: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				2: {EOF, ARITHMETIC, COMPARISION},
				3: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				4: {ARITHMETIC, COMPARISION, EOF},
				5: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				6: {ARITHMETIC, COMPARISION, EOF},
			},
		},
		{
			name:        "Unary expression - metrics",
			inputString: "-foo",
			expectedTypesFromParsePosMap: map[int][]TokenType{
				0: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN, UNARY_OP},
				1: {METRIC_ID, NUM, AGGR_OP, FUNCTION_SCALAR_ID, FUNCTION_VECTOR_ID, LEFT_PAREN},
				2: {EOF, ARITHMETIC, COMPARISION, SET, LEFT_BRACE, OFFSET_KW},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewEarleyParser(*promQLGrammar)
			chart := p.Parse(tc.inputString)
			for pos, types := range tc.expectedTypesFromParsePosMap {
				validTypes := chart.GetValidTerminalTypesAtStateSet(pos)
				var tknTypes []TokenType
				for _, ct := range validTypes {
					tknTypes = append(tknTypes, ct.TokenType)
				}

				if !isEqualTypes(tknTypes, types) {
					t.Errorf("Position %d: Got %v, expected %v\n", pos, tknTypes, types)
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
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
		},
		{
			"new input is empty",
			"sum(metric_name_one",
			"",
			[]TokenType{METRIC_ID, NUM, AGGR_OP, FUNCTION_VECTOR_ID, FUNCTION_SCALAR_ID, LEFT_PAREN, UNARY_OP},
		},
		{
			"previous input is empty",
			"",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
		},
		{
			"previous input and new input are different from beginning",
			"metric_name{label=",
			"sum(metric_name_one",
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
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
			[]TokenType{RIGHT_PAREN, LEFT_BRACE, OFFSET_KW, COMPARISION, ARITHMETIC, SET},
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

func isEqualTypes(actual interface{}, expected interface{}) bool {
	return newStringSet(actual.([]TokenType)...).Equal(newStringSet(expected.([]TokenType)...))
}
