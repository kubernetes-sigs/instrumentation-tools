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
	Expression         = NewNonTerminal("expression", true)
	MetricExpression   = NewNonTerminal("metric-expression", false)
	AggrExpression     = NewNonTerminal("aggr-expression", false)
	LabelExpression    = NewNonTerminal("label-expression", false)
	FuncCallExpression = NewNonTerminal("func-call-expression", false)
	FuncArgs           = NewNonTerminal("func-args", false)
	BinaryExpression   = NewNonTerminal("binary-expression", false)
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
		NewRule(MetricExpression, MetricIdentifier),
		// 2) a metric expression can optionally have a label expression
		NewRule(MetricExpression, MetricIdentifier, LabelExpression),

		// AGGR EXPRESSIONS:
		// 1) a metric expression can consist solely of a metric tokenType
		NewRule(AggrExpression, AggregatorOp, FuncCallExpression),
		NewRule(AggrExpression, AggregatorOp, AggregateKeyword, FuncCallExpression, FuncCallExpression),

		NewRule(FuncCallExpression, LParen, MetricExpression, RParen),
		// the commented rule below is only valid for actual funcs and grouping labels
		//NewRule(FuncCallExpression, LParen, FuncArgs, RParen),
		NewRule(FuncArgs, FuncArgs, Comma, Identifier),
		NewRule(FuncArgs, Identifier),
		// LABEL EXPRESSIONS:
		NewRule(LabelExpression, LBrace, MetricLabelIdentifier, Operator, Str, RBrace),

		// BINARY EXPRESSIONS:
		NewRule(BinaryExpression, BinaryExpression, Arithmetic, Num),
		NewRule(BinaryExpression, Num, Arithmetic, Num),
	)

	PromQLParser = newEarleyParser(*promQLGrammar)
)
