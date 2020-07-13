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
	"github.com/golang/protobuf/proto"
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

func TestSuggestedTypes(t *testing.T) {
	testCases := []struct {
		name          string
		inputString   string
		tokenPos      int
		expectedTypes []TokenType
		expectedMetric *string
	}{
		{
			name:          "If we've consumed zero tokens, then we should suggest",
			inputString:   "blah",
			tokenPos:      0,
			expectedTypes: []TokenType{METRIC_ID, NUM, AGGR_OP},
		},
		{
			name:          "If we have an empty string, then we should suggest",
			inputString:   "",
			tokenPos:      0,
			expectedTypes: []TokenType{METRIC_ID, NUM, AGGR_OP},
		},
		{
			name:          "If we've consumed zero tokens, we should suggest an ID or Num",
			inputString:   "123 + 4 + 10",
			tokenPos:      0,
			expectedTypes: []TokenType{METRIC_ID, NUM, AGGR_OP},
		},
		{
			name:          "If we detect a num we, should only suggest an arithmetic operator",
			inputString:   "123 + 4 + 10",
			tokenPos:      1,
			expectedTypes: []TokenType{ARITHMETIC},
		},
		{
			name:          "If we detect an arithmetic operator, we should suggest a num",
			inputString:   "123 + 4 + 10",
			tokenPos:      2,
			expectedTypes: []TokenType{NUM},
		},
		{
			name:          "Having consumed an ID, we should recommend a brace (paren in the future though) and EOF",
			inputString:   "metric_name{label=",
			tokenPos:      1,
			expectedTypes: []TokenType{EOF, LEFT_BRACE},
			expectedMetric: proto.String("metric_name"),
		},

		{
			name:          "Having consumed an left brace, we should recommend an ID",
			inputString:   "metric_name{label=",
			tokenPos:      2,
			expectedTypes: []TokenType{METRIC_LABEL_SUBTYPE},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "Having consumed a label, we should recommend an OPERATOR",
			inputString:   "metric_name{label=",
			tokenPos:      3,
			expectedTypes: []TokenType{OPERATOR},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "Having consumed an operator, we should recommend a string",
			inputString:   "metric_name{label=",
			tokenPos:      4,
			expectedTypes: []TokenType{STRING},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "Having consumed a labelvalue, we should recommend a closing brace",
			inputString:   "metric_name{label='asdf'",
			tokenPos:      5,
			expectedTypes: []TokenType{RIGHT_BRACE},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "If we detect an aggrgation op, we should suggest an opening paren",
			inputString:   "sum(",
			tokenPos:      1,
			expectedTypes: []TokenType{AGGR_KW, LEFT_PAREN},
		},
		{
			name:          "If we detect an aggregation op and opening paren, we should suggest an id",
			inputString:   "sum(somemetricname",
			tokenPos:      2,
			expectedTypes: []TokenType{METRIC_ID},
		},
		{
			name:          "If we detect an aggregation op and opening paren, we should close the func or label select",
			inputString:   "sum(metric_name",
			tokenPos:      3,
			expectedTypes: []TokenType{RIGHT_PAREN, LEFT_BRACE},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "should return metric label",
			inputString:   "sum(metric_name_one{",
			tokenPos:      4,
			expectedTypes: []TokenType{METRIC_LABEL_SUBTYPE},
			expectedMetric: proto.String("metric_name_one"),
		},
		{
			name:          "should return metric name",
			inputString:   "sum(metric_name",
			tokenPos:      2,
			expectedTypes: []TokenType{METRIC_ID},
		},
		{
			name:          "should return metric label subtype",
			inputString:   "metric_name{",
			tokenPos:      2,
			expectedTypes: []TokenType{METRIC_LABEL_SUBTYPE},
			expectedMetric: proto.String("metric_name"),
		},
		{
			name:          "should return metric label subtype",
			inputString: `sum(metric_name_one{`,
			tokenPos:      4,
			expectedTypes: []TokenType{METRIC_LABEL_SUBTYPE},
			expectedMetric: proto.String("metric_name_one"),
		},
		{
			name:         "suggests aggr kw",
			inputString: `sum(metric_name_one)`,
			tokenPos:      4,
			expectedTypes: []TokenType{AGGR_KW},
			//expectedMetric: proto.String("metric_name_one"),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewEarleyParser(*promQLGrammar)
			chart := p.Parse(tc.inputString)
			validTypes := chart.GetValidTerminalTypesAtStateSet(tc.tokenPos)
			var tknTypes []TokenType
			var ctxes []completionContext
			var metricName *string
			for _, ct := range validTypes {
				tknTypes = append(tknTypes, ct.TokenType)
				if ct.ctx != nil {
					if ct.ctx.metric != nil {
						metricName = ct.ctx.metric
					}
					ctxes = append(ctxes, *ct.ctx)
				}
			}
			if !reflect.DeepEqual(tc.expectedMetric, metricName) {
				t.Errorf("Got %v, Expected metric in context :%v ",
					safeRead(metricName),
					safeRead(tc.expectedMetric))
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