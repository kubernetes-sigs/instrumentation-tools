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
	"time"

	"sigs.k8s.io/instrumentation-tools/notstdlib/sets"
	"sigs.k8s.io/instrumentation-tools/promq/autocomplete"
	"sigs.k8s.io/instrumentation-tools/promq/prom"
)

func TestGetPrefix(t *testing.T) {
	testCases := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "get prefix from empty string",
			query: "",
			want:  "",
		},
		{
			name:  "'asdfsdfa{fff' should have 'fff' as a prefix",
			query: "asdfsdfa{fff",
			want:  "fff",
		},
		{
			name:  "'asdfsdfa{fff=' should have '' as a prefix",
			query: "asdfsdfa{fff=",
			want:  "",
		},
		{
			name:  "'sum(metric_name_one{' should have '' as a prefix",
			query: "sum(metric_name_one{",
			want:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := getPrefix(tc.query); got != tc.want {
				t.Errorf("getPrefix() = %v, want %v", got, tc.want)
			}
		})
	}
}

var (
	initialMetricsString = `
# HELP metric_name_one [STABLE] counter help
# TYPE metric_name_one counter
metric_name_one{dima="1",dimb="1"} 2
metric_name_one{dima="3",dimb="3"} 1
metric_name_one{dima="3",dimb="3"} 1
# HELP metric_name_two [STABLE] counter help
# TYPE metric_name_two counter
metric_name_two{dima="a",dim2="asdf"} 2
metric_name_two{dima="ba",dim2="asdf"} 2
`
)

func TestEndToEndAutoCompletion(t *testing.T) {
	index := NewTestIndex()
	index.LoadMetrics(initialMetricsString, time.Now())
	testCases := []struct {
		desc                    string
		expectedMatchesQueryMap map[string][]sets.String
	}{
		{
			desc: "completes on empty string",
			expectedMatchesQueryMap: map[string][]sets.String{
				"": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(unaryOperators)},
			},
		},
		{
			desc: "complete on binary expression - scalar binary with arithmetic operation",
			expectedMatchesQueryMap: map[string][]sets.String{
				"123 ": {sets.StringKeySet(arithmeticOperators), sets.StringKeySet(comparisionOperators)},
				"123 +": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"123 + 4 ": {sets.StringKeySet(arithmeticOperators), sets.StringKeySet(comparisionOperators)},
			},
		},
		{
			desc: "complete on binary expression - with unary expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"123 +": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"123 + (": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(unaryOperators),
				},
				"123 + (-": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"123 + (-4 ": {sets.StringKeySet(arithmeticOperators), sets.StringKeySet(comparisionOperators)},
				"123 + (-4)": {sets.StringKeySet(arithmeticOperators), sets.StringKeySet(comparisionOperators)},
			},
		},
		{
			desc: "complete on binary expression - scalar binary with comparision operation",
			expectedMatchesQueryMap: map[string][]sets.String{
				"123 + 4 <": {
					sets.NewString("metric_name_one", "metric_name_two", "bool"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"123 + 4 <=": {
					sets.NewString("metric_name_one", "metric_name_two", "bool"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"123 + 4 <= boo": {
					sets.NewString("bool"),
				},
				"123 + 4 <= bool ": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
			},
		},
		{
			desc: "complete on binary expression - no suggestion because set operators only apply between two vectors",
			expectedMatchesQueryMap: map[string][]sets.String{
				"123 and 3": {},
			},
		},
		{
			desc: "complete on binary expression - vector binary with set operation",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one ": {
					sets.NewString("offset"),
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(setOperators),
					sets.StringKeySet(comparisionOperators),
				},
				"metric_name_one an": {
					sets.NewString("and"),
				},
				"metric_name_one{dima='1'} and ": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(groupKeywords),
				},
				"metric_name_one{dima='1'} and metric_name_two{": {
					sets.NewString("dima", "dim2"),
				},
			},
		},
		{
			desc: "complete on binary expression - one_to_one vector match with arithmetic operator",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one * o": {
					sets.NewString("on"),
				},
				"metric_name_one * on(": {
					sets.NewString("dima", "dimb"),
				},
				"metric_name_one * on(dima,": {
					sets.NewString("dima", "dimb"),
				},
				"metric_name_one * on(dima,) m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
				"metric_name_one * on(dima,) g": {
					sets.NewString("group_right", "group_left"),
				},
				"metric_name_one * on(dima,) metric_name_two{": {
					sets.NewString("dima", "dim2"),
				},
				"metric_name_one * on(dima,) metric_name_two o": {
					sets.NewString("offset", "or"),
				},
			},
		},
		{
			desc: "complete on binary expression - one_to_one vector match with set operator",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one a": {
					sets.NewString("and"),
				},
				"metric_name_one and o": {
					sets.NewString("on"),
				},
				"metric_name_one and on(": {
					sets.NewString("dima", "dimb"),
				},
				"metric_name_one and on(dima,) m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
				"metric_name_one and on(dima,) g": {
					sets.NewString(),
				},
			},
		},
		{
			desc: "complete on binary expression - one_to_many vector match",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one / on(dima,dima) g": {
					sets.NewString("group_right", "group_left"),
				},
				"metric_name_one / on(dima,dima) group_left(d": {
					sets.NewString("dima", "dimb", "day_of_month", "day_of_week", "days_in_month", "delta", "deriv"),
				},
				"metric_name_one / on(dima,dima) group_left(m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
				"metric_name_one / on(dima,dima) group_left(metric_name_two o": {
					sets.NewString("offset", "or"),
				},
				"metric_name_one / on(dima,dima) group_left(dima) m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
			},
		},
		{
			desc: "complete on metric expression - metric name",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
				"metric_name_one o": {
					sets.NewString("offset", "or"),
				},
			},
		},
		{
			desc: "complete on metric expression - with labels",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one{": {
					sets.NewString("dima", "dimb"),
				},
				"metric_name_one{dima=": {
					sets.NewString("\"1\"", "\"3\""),
				},
			},
		},
		{
			desc: "complete on metric expression - with offset",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one offset 5": {
					sets.StringKeySet(timeUnits),
				},
			},
		},
		{
			desc: "complete on metric expression - range vector selector",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one[": {},
				"metric_name_one[3": {
					sets.StringKeySet(timeUnits),
				},
				"metric_name_one[3m]": {
					sets.NewString("offset"),
				},
			},
		},
		{
			desc: "complete on aggregation expression - the clause is before expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"su": {
					sets.NewString("sum", "sum_over_time"),
				},
				"sum ": {
					sets.StringKeySet(aggregateKeywords),
				},
				"sum by (": {
					sets.NewString(),
				},
				"sum by (dima) (me": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
			},
		},
		{
			desc: "complete on aggregation expression - multiple label matchers",
			expectedMatchesQueryMap: map[string][]sets.String{
				"sum(metric_name_": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
				"sum(metric_name_one{": {
					sets.NewString("dima", "dimb"),
				},
				"sum(metric_name_one{dima=": {
					sets.NewString("\"1\"", "\"3\""),
				},
				"sum(metric_name_one{dima='1'} ": {
					sets.NewString("offset"),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
					sets.StringKeySet(arithmeticOperators),
				},
				"sum(metric_name_one{dima='1'})": {
					sets.StringKeySet(aggregateKeywords),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
					sets.StringKeySet(arithmeticOperators),
				},
			},
		},
		{
			desc: "complete on aggregation expression - the clause is after expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"sum(metric_name_one{dima='1'}) b": {
					sets.NewString("by"),
				},
				"sum(metric_name_one{dima='1'}) by (": {
					sets.NewString("dima", "dimb"),
				},
				"sum(metric_name_one{dima='1'}) by (dima)": {
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
					sets.StringKeySet(arithmeticOperators),
				},
			},
		},
		{
			desc: "complete on function expression - scalar function",
			expectedMatchesQueryMap: map[string][]sets.String{
				"sca": {
					sets.NewString("scalar"),
				},
				"scalar(": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(unaryOperators),
				},
				"scalar(me": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
				"scalar(metric_name_one)": {
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(arithmeticOperators),
				},
			},
		},
		{
			desc: "complete on function expression - have expression as arg",
			expectedMatchesQueryMap: map[string][]sets.String{
				"floor(metric_name_one{": {
					sets.NewString("dima", "dimb"),
				},
				"floor(metric_name_one{dima=": {
					sets.NewString("\"1\"", "\"3\""),
				},
				"floor(metric_name_one{dima='1'}": {
					sets.NewString("offset"),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(setOperators),
				},
				"floor(metric_name_one{dima='1'})": {
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(setOperators),
				},
			},
		},
		{
			desc: "complete on function expression - have aggregation expression as arg",
			expectedMatchesQueryMap: map[string][]sets.String{
				"vector(su": {
					sets.NewString("sum", "sum_over_time"),
				},
				"vector(sum(me": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
				"vector(sum(metric_name_one)": {
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(setOperators),
					sets.StringKeySet(aggregateKeywords),
				},
				"vector(sum(metric_name_one))": {
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(setOperators),
				},
			},
		},
		{
			desc: "complete on function expression - have multiple args",
			expectedMatchesQueryMap: map[string][]sets.String{
				"round(metric_name_one, ": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(unaryOperators),
				},
				"round(metric_name_one, -": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
				},
				"round(metric_name_one, -5 ": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
				},
			},
		},
		{
			desc: "complete on function expression - nested function call",
			expectedMatchesQueryMap: map[string][]sets.String{
				"ceil(ab": {
					sets.NewString("abs", "absent", "absent_over_time"),
				},
			},
		},
		{
			desc: "complete on subquery expression - expr is vectorSelector",
			expectedMatchesQueryMap: map[string][]sets.String{
				"metric_name_one{dima='1'}[": {},
				"metric_name_one{dima='1'}[10": {
					sets.StringKeySet(timeUnits),
				},
				"metric_name_one{dima='1'}[10m:": {},
				"metric_name_one{dima='1'}[10m:6": {
					sets.StringKeySet(timeUnits),
				},
				"metric_name_one{dima='1'}[10m:6s]": {
					sets.NewString("offset"),
				},
			},
		},
		{
			desc: "complete on subquery expression - expr is function expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"rate(metric_name_one{dima='1'}[5m])": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
				},
				"rate(metric_name_one{dima='1'}[5m])[": {},
				"rate(metric_name_one{dima='1'}[5m])[10": {
					sets.StringKeySet(timeUnits),
				},
				"rate(metric_name_one{dima='1'}[5m])[10m:": {},
				"rate(metric_name_one{dima='1'}[5m])[10m:6s]": {
					sets.NewString("offset"),
				},
			},
		},
		{
			desc: "complete on parentheses expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"((metric_name_one{": {
					sets.NewString("dima", "dimb"),
				},
				"((metric_name_one + metric_name_two{": {
					sets.NewString("dima", "dim2"),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima=": {
					sets.NewString("\"a\"", "\"ba\""),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima='a'}": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
					sets.NewString("offset"),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima='a'})": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima='a'}) + m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima='a'}) + metric_name_one)": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
					sets.StringKeySet(setOperators),
				},
				"((metric_name_one{dima='1'} + metric_name_two{dima='a'}) + metric_name_one) - ": {
					sets.NewString("metric_name_one", "metric_name_two"),
					sets.StringKeySet(aggregators),
					sets.StringKeySet(scalarFunctions),
					sets.StringKeySet(vectorFunctions),
					sets.StringKeySet(groupKeywords),
				},
			},
		},
		{
			desc: "complete on unary expression",
			expectedMatchesQueryMap: map[string][]sets.String{
				"-me": {
					sets.NewString("metric_name_one", "metric_name_two"),
				},
				"-1 ": {
					sets.StringKeySet(arithmeticOperators),
					sets.StringKeySet(comparisionOperators),
				},
				"-m": {
					sets.NewString("metric_name_one", "metric_name_two", "max_over_time", "min_over_time", "minute", "month", "max", "min"),
				},
				"-s": {
					sets.NewString("sum", "scalar", "sort", "sort_desc", "sqrt", "stddev", "stddev_over_time", "stdvar", "stdvar_over_time", "sum_over_time"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewPromQLCompleter(index)
			for query, expectedMatches := range tc.expectedMatchesQueryMap {
				matches := c.GenerateSuggestions(query, len(query))
				matchVals := toSet(matches)
				expectedVals := union(expectedMatches...)
				if !reflect.DeepEqual(matchVals, expectedVals) {
					t.Errorf("Query %v: got %v matches [%v]\n expected %v\n", query, len(matchVals), matchVals, expectedVals)
				}
			}

		})
	}
}

func toSet(matches []autocomplete.Match) sets.String {
	ret := sets.NewString()
	for _, m := range matches {
		ret.Insert(m.GetValue())
	}
	return ret
}

func union(strSets ...sets.String) sets.String {
	ss := sets.String{}
	for _, s := range strSets {
		ss = ss.Union(s)
	}
	return ss
}

type TestIndex struct {
	prom.Indexer
}

func (ti *TestIndex) LoadMetrics(rawMetricsText string, time time.Time) error {
	parsedSeries, err := prom.ParseTextData([]byte(rawMetricsText), time)
	if err != nil {
		return err
	}
	for _, series := range parsedSeries {
		ti.UpdateMetric(series)
	}
	return nil
}

func NewTestIndex() *TestIndex {
	return &TestIndex{prom.NewIndex()}
}
