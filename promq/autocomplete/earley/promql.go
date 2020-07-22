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

type terminalType string

var (
	// non-terminals
	Expression           = NewNonTerminal("expression", true)
	MetricExpression     = NewNonTerminal("metric-expression", false)
	AggrExpression       = NewNonTerminal("aggr-expression", false)
	LabelsExpression     = NewNonTerminal("labels-expression", false)
	LabelValueExpression = NewNonTerminal("label-value-expression", false)
	AggrCallExpression   = NewNonTerminal("aggr-call-expression", false)
	MetricLabelArgs      = NewNonTerminal("func-args", false)
	BinaryExpression     = NewNonTerminal("binary-expression", false)
	//AggrFuncParam   = NewNonTerminal("func-param", false) // sometimes optional, but sometimes necessary

	// terminals
	Identifier            = NewTerminal(ID)                                  // this one is ambiguous
	MetricIdentifier      = NewTerminalWithSubType(ID, METRIC_ID)            // this one is ambiguous
	MetricLabelIdentifier = NewTerminalWithSubType(ID, METRIC_LABEL_SUBTYPE) // this one is ambiguous
	AggregatorOp          = NewTerminal(AGGR_OP)
	AggregateKeyword      = NewTerminal(AGGR_KW)
	Arithmetic            = NewTerminal(ARITHMETIC) // we don't give a shit about precedence
	Operator              = NewTerminal(OPERATOR)

	LBrace = NewTerminal(LEFT_BRACE)
	RBrace = NewTerminal(RIGHT_BRACE)
	Comma  = NewTerminal(COMMA)
	LParen = NewTerminal(LEFT_PAREN)
	RParen = NewTerminal(RIGHT_PAREN)
	Str    = NewTerminal(STRING)
	Num    = NewTerminal(NUM)
	Eof    = NewTerminal(EOF)

	promQLGrammar = NewGrammar(

		// TOP LEVEL RULES:

		// 1) an expression can be a metric/binary/aggr expression
		NewRule(Expression, MetricExpression, Eof),
		NewRule(Expression, BinaryExpression, Eof),
		NewRule(Expression, AggrExpression, Eof),

		// METRIC EXPRESSIONS:
		// 1) a metric expression can consist solely of a metric tokenType
		NewRule(MetricExpression, MetricIdentifier),
		// 2) a metric expression can optionally have a label expression
		NewRule(MetricExpression, MetricIdentifier, LabelValueExpression),

		// AGGR EXPRESSIONS:
		// 1) a aggregation operation expression can consist solely of a metric tokenType
		// sum(metric) by (label1)
		NewRule(AggrExpression, AggregatorOp, AggrCallExpression, AggregateKeyword, LabelsExpression),
		// sum by (label) (metric)
		NewRule(AggrExpression, AggregatorOp, AggregateKeyword, LabelsExpression, AggrCallExpression),
		// '(metric{label="blah"})'
		NewRule(AggrCallExpression, LParen, MetricExpression, RParen),

		// LABEL EXPRESSIONS:
		NewRule(LabelsExpression, LParen, MetricLabelArgs, RParen),
		// todo(han) it is also valid to have multiple targeted additional metric label
		// todo(han) i.e. sum(metricname{label1="blah",label2="else"}) by (label3)
		NewRule(MetricLabelArgs, MetricLabelArgs, Comma, MetricLabelIdentifier),
		NewRule(MetricLabelArgs, MetricLabelIdentifier),

		NewRule(LabelValueExpression, LBrace, MetricLabelIdentifier, Operator, Str, RBrace),

		// BINARY EXPRESSIONS:
		NewRule(BinaryExpression, BinaryExpression, Arithmetic, Num),
		NewRule(BinaryExpression, Num, Arithmetic, Num),
	)

	PromQLParser = NewEarleyParser(*promQLGrammar)

	aggregators = map[string]string{
		"sum":          "calculate sum over dimensions",
		"max":          "select maximum over dimensions",
		"min":          "select minimum over dimensions",
		"avg":          "calculate the average over dimensions",
		"stddev":       "calculate population standard deviation over dimensions",
		"stdvar":       "calculate population standard variance over dimensions",
		"count":        "count number of elements in the vector",
		"count_values": "count number of elements with the same value",
		"bottomk":      "smallest k elements by sample value",
		"topk":         "largest k elements by sample value",
		"quantile":     "calculate φ-quantile (0 ≤ φ ≤ 1) over dimensions",
	}

	// Todo:(yuchen) add the description for aggr_kw
	aggregateKeywords = map[string]string{
		"offset":      "",
		"by":          "",
		"without":     "",
		"on":          "",
		"ignoring":    "",
		"group_left":  "",
		"group_right": "",
		"bool":        "",
	}
)
