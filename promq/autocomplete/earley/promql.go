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
	Root               = NewNonTerminal("root", true)
	Expression         = NewNonTerminal("expression", false)
	MetricExpression   = NewNonTerminal("metric-expression", false)
	AggrExpression     = NewNonTerminal("aggr-expression", false)
	BinaryExpression   = NewNonTerminal("binary-expression", false)
	FuncExpression     = NewNonTerminal("function-expression", false)
	SubqueryExpression = NewNonTerminal("subquery-expression", false)
	NumLiteral         = NewNonTerminal("num-literal", false)
	StrLiteral         = NewNonTerminal("str-literal", false)

	VectorFuncExpression = NewNonTerminal("vector-function-expression", false)
	ScalarFuncExpression = NewNonTerminal("scalar-function-expression", false)

	MatrixSelector = NewNonTerminal("matrix-selector", false)
	VectorSelector = NewNonTerminal("vector-selector", false)

	LabelsExpression      = NewNonTerminal("labels-expression", false)
	LabelsMatchExpression = NewNonTerminal("labels-match-expression", false)
	LabelValueExpression  = NewNonTerminal("label-value-expression", false)
	AggrCallExpression    = NewNonTerminal("aggr-call-expression", false)
	MetricLabelArgs       = NewNonTerminal("label-args", false)

	OffsetExpression = NewNonTerminal("offset-expression", false)
	//AggrFuncParam   = NewNonTerminal("func-param", false) // sometimes optional, but sometimes necessary

	FunctionCallBody = NewNonTerminal("function-call-body", false)
	FunctionCallArgs = NewNonTerminal("function-call-args", false)

	// Expression types:
	ScalarTypeExpression   = NewNonTerminal("scalar-type-expression", false)
	VectorTypeExpression   = NewNonTerminal("vector-type-expression", false)
	MatrixTypeExpression   = NewNonTerminal("matrix-type-expression", false)
	ScalarBinaryExpression = NewNonTerminal("scalar-binary-expression", false)
	VectorBinaryExpression = NewNonTerminal("vector-binary-expression", false)

	// Binary expressions related non-terminals:
	BinaryOperator      = NewNonTerminal("scalar-binary-operator", false)
	BinaryGroupModifier = NewNonTerminal("binary-group-modifier", false)

	// terminals
	Identifier               = NewTerminal(ID)                                  // this one is ambiguous
	MetricIdentifier         = NewTerminalWithSubType(ID, METRIC_ID)            // this one is ambiguous
	MetricLabelIdentifier    = NewTerminalWithSubType(ID, METRIC_LABEL_SUBTYPE) // this one is ambiguous
	ScalarFunctionIdentifier = NewTerminalWithSubType(ID, FUNCTION_SCALAR_ID)
	VectorFunctionIdentifier = NewTerminalWithSubType(ID, FUNCTION_VECTOR_ID)

	AggregatorOp     = NewTerminal(AGGR_OP)
	AggregateKeyword = NewTerminal(AGGR_KW)
	BoolKeyword      = NewTerminalWithSubType(KEYWORD, BOOL_KW)
	OffsetKeyword    = NewTerminalWithSubType(KEYWORD, OFFSET_KW)
	GroupKeyword     = NewTerminal(GROUP_KW)
	GroupSide        = NewTerminal(GROUP_SIDE)

	Operator           = NewTerminal(OPERATOR)
	Arithmetic         = NewTerminal(ARITHMETIC)
	SetOperator        = NewTerminal(SET)
	LabelMatchOperator = NewTerminalWithSubType(OPERATOR, LABELMATCH)
	Comparision        = NewTerminalWithSubType(OPERATOR, COMPARISION)

	LBrace   = NewTerminal(LEFT_BRACE)
	RBrace   = NewTerminal(RIGHT_BRACE)
	LBracket = NewTerminal(LEFT_BRACKET)
	RBracket = NewTerminal(RIGHT_BRACKET)
	Comma    = NewTerminal(COMMA)
	Colon    = NewTerminal(COLON)
	LParen   = NewTerminal(LEFT_PAREN)
	RParen   = NewTerminal(RIGHT_PAREN)
	Str      = NewTerminal(STRING)
	Num      = NewTerminal(NUM)
	Duration = NewTerminal(DURATION)
	Eof      = NewTerminal(EOF)

	promQLGrammar = NewGrammar(

		//START RULE:
		NewRule(Root, Expression, Eof),
		// TOP LEVEL RULES:
		// 1) an expression can be a scalar/vector/matrix type expression
		NewRule(Expression, ScalarTypeExpression),
		NewRule(Expression, VectorTypeExpression),
		NewRule(Expression, MatrixTypeExpression),

		// EXPRESSION TYPE:
		// 1) scalar type expression
		NewRule(ScalarTypeExpression, ScalarBinaryExpression),
		NewRule(ScalarTypeExpression, Num),
		NewRule(ScalarTypeExpression, ScalarFuncExpression),
		// 2) vector type expression
		NewRule(VectorTypeExpression, VectorSelector),
		NewRule(VectorTypeExpression, VectorBinaryExpression),
		NewRule(VectorTypeExpression, VectorFuncExpression),
		NewRule(VectorTypeExpression, AggrExpression),
		// 3) matrix type expression
		NewRule(MatrixTypeExpression, MatrixSelector),
		NewRule(MatrixTypeExpression, SubqueryExpression),

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
		NewRule(AggrCallExpression, LParen, VectorTypeExpression, RParen),

		// LABEL EXPRESSIONS:
		NewRule(LabelsExpression, LParen, MetricLabelArgs, RParen),
		// label list could be empty
		NewRule(LabelsExpression, LParen, RParen),
		// label list can end with comma
		NewRule(LabelsExpression, LParen, MetricLabelArgs, Comma, RParen),

		// todo(han) it is also valid to have multiple targeted additional metric label
		// todo(han) i.e. sum(metricname{label1="blah",label2="else"}) by (label3)
		NewRule(MetricLabelArgs, MetricLabelArgs, Comma, MetricLabelIdentifier),
		NewRule(MetricLabelArgs, MetricLabelIdentifier),

		// {label1="blah",label2="else"}
		NewRule(LabelsMatchExpression, LBrace, LabelValueExpression, RBrace),
		NewRule(LabelValueExpression, MetricLabelIdentifier, LabelMatchOperator, Str),
		NewRule(LabelValueExpression, LabelValueExpression, Comma, MetricLabelIdentifier, LabelMatchOperator, Str),

		// BINARY EXPRESSIONS:
		// 1) scalar type binary expr: both left and right are scalar type
		NewRule(BinaryExpression, ScalarBinaryExpression),
		// 2) vector type binary expr
		NewRule(BinaryExpression, VectorBinaryExpression),
		// binary expression can be embraced by parenthesis
		NewRule(ScalarBinaryExpression, LParen, ScalarBinaryExpression, RParen),
		NewRule(VectorBinaryExpression, LParen, VectorBinaryExpression, RParen),

		// Binary Operators:
		NewRule(BinaryOperator, Arithmetic),
		NewRule(BinaryOperator, Comparision),
		NewRule(BinaryOperator, Comparision, BoolKeyword),

		// Binary group modifiers:
		NewRule(BinaryGroupModifier, GroupKeyword, LabelsExpression),
		NewRule(BinaryGroupModifier, GroupKeyword, LabelsExpression, GroupSide),
		NewRule(BinaryGroupModifier, GroupKeyword, LabelsExpression, GroupSide, LabelsExpression),

		NewRule(ScalarBinaryExpression, ScalarTypeExpression, Arithmetic, ScalarTypeExpression),
		NewRule(ScalarBinaryExpression, ScalarTypeExpression, Comparision, BoolKeyword, ScalarTypeExpression),

		NewRule(VectorBinaryExpression, ScalarTypeExpression, BinaryOperator, VectorTypeExpression),
		NewRule(VectorBinaryExpression, VectorTypeExpression, BinaryOperator, ScalarTypeExpression),
		NewRule(VectorBinaryExpression, VectorTypeExpression, BinaryOperator, VectorTypeExpression),
		NewRule(VectorBinaryExpression, VectorTypeExpression, SetOperator, VectorTypeExpression),
		NewRule(VectorBinaryExpression, VectorTypeExpression, BinaryOperator, BinaryGroupModifier, VectorTypeExpression),
		// Set operations match with all possible entries in the right vector by default.
		NewRule(VectorBinaryExpression, VectorTypeExpression, SetOperator, GroupKeyword, LabelsExpression, VectorTypeExpression),

		// FUNCTION EXPRESSIONS:
		// Todo(yuchen) The input args can vary from different functions. Here I only separate the function with different return type.
		NewRule(FuncExpression, ScalarFuncExpression),
		NewRule(FuncExpression, VectorTypeExpression),

		// the functions that return vector type expression
		NewRule(VectorFuncExpression, VectorFunctionIdentifier, FunctionCallBody),
		NewRule(FunctionCallBody, LParen, RParen),
		NewRule(FunctionCallBody, LParen, FunctionCallArgs, RParen),
		NewRule(FunctionCallArgs, Expression),
		NewRule(FunctionCallArgs, FunctionCallArgs, Comma, Expression),
		// the functions that return scalar type expression: time() scalar(vector)
		NewRule(ScalarFuncExpression, ScalarFunctionIdentifier, LParen, RParen),
		NewRule(ScalarFuncExpression, ScalarFunctionIdentifier, LParen, VectorTypeExpression, RParen),

		// SUBQUERY EXPRESSIONS:
		NewRule(SubqueryExpression, VectorTypeExpression, LBracket, Duration, Colon, RBracket),
		NewRule(SubqueryExpression, VectorTypeExpression, LBracket, Duration, Colon, Duration, RBracket),
		NewRule(SubqueryExpression, VectorTypeExpression, LBracket, Duration, Colon, RBracket, OffsetExpression),
		NewRule(SubqueryExpression, VectorTypeExpression, LBracket, Duration, Colon, Duration, RBracket, OffsetExpression),
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

	groupKeywords = map[string]string{
		"ignoring": "",
		"on":       "",
	}

	groupSideKeywords = map[string]string{
		"group_left":  "",
		"group_right": "",
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

	setOperators = map[string]string{
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

	scalarFunctions = map[string]string{
		"time":   "time() returns the time at which the expression is to be evaluated in seconds ",
		"scalar": "given a single-element input vector, scalar(v instant-vector) returns the sample value of that single element as a scalar.",
	}

	vectorFunctions = map[string]string{
		"abs":                "abs(v instant-vector) returns the input vector with all sample values converted to their absolute value",
		"absent":             "absent(v instant-vector) returns an empty vector if the vector passed to it has any elements and a 1-element vector with the value 1 if the vector passed to it has no elements",
		"absent_over_time":   "absent_over_time(v range-vector) returns an empty vector if the range vector passed to it has any elements and a 1-element vector with the value 1 if the range vector passed to it has no elements",
		"avg_over_time":      "avg_over_time(v range-vector) returns the average value of all points in the specified interval",
		"ceil":               "ceil(v instant-vector) rounds the sample values of all elements in input vector up to the nearest integer",
		"changes":            "for each input time series, changes(v range-vector) returns the number of times its value has changed within the provided time range as an instant vector.",
		"clamp_max":          "clamp_max(v instant-vector, max scalar) clamps the sample values of all elements in v to have an upper limit of max",
		"clamp_min":          "clamp_min(v instant-vector, min scalar) clamps the sample values of all elements in v to have a lower limit of min",
		"count_over_time":    "count_over_time(v range-vector) returns the count of all values in the specified interval",
		"days_in_month":      "days_in_month(v=vector(time()) instant-vector) returns number of days in the month for each of the given times in UTC",
		"day_of_month":       "day_of_month(v=vector(time()) instant-vector) returns the day of the month for each of the given times in UTC",
		"day_of_week":        "day_of_week(v=vector(time()) instant-vector) returns the day of the week for each of the given times in UTC",
		"delta":              "delta(v range-vector) calculates the difference between the first and last value of each time series element in a range vector v, returning an instant vector with the given deltas and equivalent labels",
		"deriv":              "deriv(v range-vector) calculates the per-second derivative of the time series in a range vector v. deriv should only be used with gauges.",
		"exp":                "exp(v instant-vector) calculates the exponential function for all elements in v",
		"floor":              "floor(v instant-vector) rounds the sample values of all elements in v down to the nearest integer",
		"histogram_quantile": "histogram_quantile(φ float, b instant-vector) calculates the φ-quantile (0 ≤ φ ≤ 1) from the buckets b of a histogram",
		"holt_winters":       "holt_winters(v range-vector, sf scalar, tf scalar) produces a smoothed value for time series based on the range in v",
		"hour":               "hour(v=vector(time()) instant-vector) returns the hour of the day for each of the given times in UTC",
		"idelta":             "idelta(v range-vector) calculates the difference between the last two samples in the range vector v, returning an instant vector with the given deltas and equivalent labels",
		"increase":           "increase(v range-vector) calculates the increase in the time series in the range vector",
		"irate":              "irate(v range-vector) calculates the per-second instant rate of increase of the time series in the range vector",
		"label_replace":      "for each timeseries in v, label_replace(v instant-vector, dst_label string, replacement string, src_label string, regex string) matches the regular expression regex against the label src_label. If it matches, then the timeseries is returned with the label dst_label replaced by the expansion of replacement.",
		"label_join":         "for each timeseries in v, label_join(v instant-vector, dst_label string, separator string, src_label_1 string, src_label_2 string, ...) joins all the values of all the src_labels using separator and returns the timeseries with the label dst_label containing the joined value.",
		"ln":                 "ln(v instant-vector) calculates the natural logarithm for all elements in v",
		"log10":              "log10(v instant-vector) calculates the decimal logarithm for all elements in v",
		"log2":               "log2(v instant-vector) calculates the binary logarithm for all elements in v",
		"max_over_time":      "max_over_time(range-vector) returns the maximum value of all points in the specified interval",
		"min_over_time":      "min_over_time(range-vector) returns the minimum value of all points in the specified interval",
		"minute":             "minute(v=vector(time()) instant-vector) returns the minute of the hour for each of the given times in UTC",
		"month":              "month(v=vector(time()) instant-vector) returns the month of the year for each of the given times in UTC",
		"predict_linear":     "predict_linear(v range-vector, t scalar) predicts the value of time series t seconds from now, based on the range vector v",
		"quantile_over_time": "quantile_over_time(scalar, range-vector) returns the φ-quantile (0 ≤ φ ≤ 1) of the values in the specified interval",
		"rate":               "rate(v range-vector) calculates the per-second average rate of increase of the time series in the range vector",
		"resets":             "for each input time series, resets(v range-vector) returns the number of counter resets within the provided time range as an instant vector",
		"round":              "round(v instant-vector, to_nearest=1 scalar) rounds the sample values of all elements in v to the nearest integer",
		"sort":               "sort(v instant-vector) returns vector elements sorted by their sample values, in ascending order",
		"sort_desc":          "sort(v instant-vector) returns vector elements sorted by their sample values, in descending order",
		"sqrt":               "sqrt(v instant-vector) calculates the square root of all elements in v",
		"stddev_over_time":   "stddev_over_time(range-vector) returns the population standard deviation of the values in the specified interval",
		"stdvar_over_time":   "stdvar_over_time(range-vector) returns the population standard variance of the values in the specified interval",
		"sum_over_time":      "sum_over_time(range-vector) returns the sum of all values in the specified interval",
		"timestamp":          "timestamp(v instant-vector) returns the timestamp of each of the samples of the given vector as the number of seconds",
		"vector":             "vector(s scalar) returns the scalar s as a vector with no labels",
		"year":               "year(v=vector(time()) instant-vector) returns the year for each of the given times in UTC",
	}
)
