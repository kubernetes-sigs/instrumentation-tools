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
	Expression       = NewNonTerminal("expression", true)
	MetricExpression = NewNonTerminal("metric-expression", false)
	AggrExpression   = NewNonTerminal("aggr-expression", false)
	BinaryExpression = NewNonTerminal("binary-expression", false)
	FuncExpression   = NewNonTerminal("function-expression", false)

	MatrixSelector = NewNonTerminal("matrix-selector", false)
	VectorSelector = NewNonTerminal("vector-selector", false)

	LabelsExpression      = NewNonTerminal("labels-expression", false)
	LabelsMatchExpression = NewNonTerminal("labels-match-expression", false)
	LabelValueExpression  = NewNonTerminal("label-value-expression", false)
	AggrCallExpression    = NewNonTerminal("aggr-call-expression", false)
	MetricLabelArgs       = NewNonTerminal("func-args", false)

	OffsetExpression = NewNonTerminal("offset-expression", false)
	//AggrFuncParam   = NewNonTerminal("func-param", false) // sometimes optional, but sometimes necessary

	FunctionCallBody = NewNonTerminal("function-call-body", false)
	FunctionCallArgs = NewNonTerminal("function-call-args", false)
	FunctionArgsType = NewNonTerminal("function-args-type", false)

	// terminals
	Identifier            = NewTerminal(ID)                                  // this one is ambiguous
	MetricIdentifier      = NewTerminalWithSubType(ID, METRIC_ID)            // this one is ambiguous
	MetricLabelIdentifier = NewTerminalWithSubType(ID, METRIC_LABEL_SUBTYPE) // this one is ambiguous
	FunctionIdentifier    = NewTerminalWithSubType(ID, FUNCTION_ID)
	AggregatorOp          = NewTerminal(AGGR_OP)

	AggregateKeyword = NewTerminal(AGGR_KW)
	BoolKeyword      = NewTerminalWithSubType(KEYWORD, BOOL_KW)
	OffsetKeyword    = NewTerminalWithSubType(KEYWORD, OFFSET_KW)

	Operator           = NewTerminal(OPERATOR)
	Arithmetic         = NewTerminal(ARITHMETIC)
	Logical            = NewTerminal(LOGICAL)
	LabelMatchOperator = NewTerminalWithSubType(OPERATOR, LABELMATCH)
	Comparision        = NewTerminalWithSubType(OPERATOR, COMPARISION)

	LBrace   = NewTerminal(LEFT_BRACE)
	RBrace   = NewTerminal(RIGHT_BRACE)
	LBracket = NewTerminal(LEFT_BRACKET)
	RBracket = NewTerminal(RIGHT_BRACKET)
	Comma    = NewTerminal(COMMA)
	LParen   = NewTerminal(LEFT_PAREN)
	RParen   = NewTerminal(RIGHT_PAREN)
	Str      = NewTerminal(STRING)
	Num      = NewTerminal(NUM)
	Duration = NewTerminal(DURATION)
	Eof      = NewTerminal(EOF)

	promQLGrammar = NewGrammar(

		// TOP LEVEL RULES:
		// 1) an expression can be a metric/binary/aggr/function expression
		NewRule(Expression, MetricExpression, Eof),
		NewRule(Expression, BinaryExpression, Eof),
		NewRule(Expression, AggrExpression, Eof),
		NewRule(Expression, FuncExpression, Eof),

		// METRIC EXPRESSIONS:
		// 1) Instant vector selectors
		NewRule(MetricExpression, VectorSelector),
		// 2) Range Vector Selectors
		NewRule(MetricExpression, MatrixSelector),

		// VECTOR SELECTOR
		// 1) a vector selector can consist solely of a metric tokenType
		NewRule(VectorSelector, MetricIdentifier),
		// 2) a vector selector can optionally have a label expression
		NewRule(VectorSelector, MetricIdentifier, LabelsMatchExpression),
		// 3) a metric expression can optionally have offset to get historical data
		NewRule(VectorSelector, MetricIdentifier, OffsetExpression),
		NewRule(VectorSelector, MetricIdentifier, LabelsMatchExpression, OffsetExpression),

		// MATRIX SELECTOR
		// metric[5m]
		NewRule(MatrixSelector, MetricIdentifier, LBracket, Duration, RBracket),
		NewRule(MatrixSelector, MetricIdentifier, LabelsMatchExpression, LBracket, Duration, RBracket),
		// metric[5m] offset 3h
		NewRule(MatrixSelector, MetricIdentifier, LBracket, Duration, RBracket, OffsetExpression),
		NewRule(MatrixSelector, MetricIdentifier, LabelsMatchExpression, LBracket, Duration, RBracket, OffsetExpression),

		// UNARY EXPRESSIONS:
		NewRule(OffsetExpression, OffsetKeyword, Duration),

		// AGGR EXPRESSIONS:
		// 1) a aggregation operation expression can consist solely of a metric tokenType
		// <aggr-op>([parameter,] <vector expression>)
		NewRule(AggrExpression, AggregatorOp, AggrCallExpression),
		// 2) <aggr-op>([parameter,] <vector expression>) [without|by (<label list>)]
		// sum(metric) by (label1)
		NewRule(AggrExpression, AggregatorOp, AggrCallExpression, AggregateKeyword, LabelsExpression),
		// 3) <aggr-op> [without|by (<label list>)] ([parameter,] <vector expression>)
		// sum by (label) (metric)
		NewRule(AggrExpression, AggregatorOp, AggregateKeyword, LabelsExpression, AggrCallExpression),
		// '(metric{label="blah"})'
		NewRule(AggrCallExpression, LParen, VectorSelector, RParen),

		// LABEL EXPRESSIONS:
		NewRule(LabelsExpression, LParen, MetricLabelArgs, RParen),
		// label list could be empty
		NewRule(LabelsExpression, LParen, RParen),
		// todo(han) it is also valid to have multiple targeted additional metric label
		// todo(han) i.e. sum(metricname{label1="blah",label2="else"}) by (label3)
		NewRule(MetricLabelArgs, MetricLabelArgs, Comma, MetricLabelIdentifier),
		NewRule(MetricLabelArgs, MetricLabelIdentifier),

		// {label1="blah",label2="else"}
		NewRule(LabelsMatchExpression, LBrace, LabelValueExpression, RBrace),
		NewRule(LabelValueExpression, MetricLabelIdentifier, LabelMatchOperator, Str),
		NewRule(LabelValueExpression, LabelValueExpression, Comma, MetricLabelIdentifier, LabelMatchOperator, Str),

		//NewRule(LabelValueExpression, LBrace, MetricLabelIdentifier, Operator, Str, RBrace),

		// BINARY EXPRESSIONS:
		// 1 + 1
		NewRule(BinaryExpression, BinaryExpression, Arithmetic, Num),
		NewRule(BinaryExpression, BinaryExpression, Comparision, BoolKeyword, Num),
		// 1 == 1
		NewRule(BinaryExpression, Num, Arithmetic, Num),
		NewRule(BinaryExpression, Num, Comparision, BoolKeyword, Num),

		//FUNCTION EXPRESSIONS:
		//Todo(yuchen) The input args can vary from different functions. Here I only define the general rule.
		NewRule(FuncExpression, FunctionIdentifier, FunctionCallBody),
		// time()
		NewRule(FunctionCallBody, LParen, RParen),
		NewRule(FunctionCallBody, LParen, FunctionCallArgs, RParen),
		NewRule(FunctionCallArgs, FunctionArgsType),
		NewRule(FunctionCallArgs, FunctionCallArgs, Comma, FunctionArgsType),

		NewRule(FunctionArgsType, MetricExpression),
		NewRule(FunctionArgsType, AggrExpression),
		NewRule(FunctionArgsType, FuncExpression),
		NewRule(FunctionArgsType, Num),
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

	// Todo:(yuchen) add the description for keywords
	aggregateKeywords = map[string]string{
		"by":      "",
		"without": "",
	}

	keywords = map[string]string{
		"bool":   "",
		"offset": "",
	}

	arithmaticOperators = map[string]string{
		"+": "",
		"-": "",
		"*": "",
		"/": "",
	}

	comparisionOperators = map[string]string{
		"==": "equal",
		"!=": "not equal",
		">":  "greater than",
		"<":  "less than",
		">=": "greater or equal",
		"<=": "less or equal",
	}

	logicalOperators = map[string]string{
		"and":    "intersection",
		"or":     "union",
		"unless": "complement",
	}

	labelMatchOperators = map[string]string{
		"=":  "match equal",
		"!=": "match not equal",
		"=~": "match regexp",
		"!~": "match not regexp",
	}

	timeUnits = map[string]string{
		"s": "seconds",
		"m": "minuets",
		"h": "hours",
		"d": "days",
		"w": "weeks",
		"y": "years",
	}
)
