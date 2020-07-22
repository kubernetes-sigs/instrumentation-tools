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
		desc            string
		query           string
		expectedMatches sets.String
	}{
		{
			desc:            "completes on metric name in aggregation context",
			query:           `sum(metric_name_o`,
			expectedMatches: sets.NewString("metric_name_one"),
		},
		{
			desc:            "completes on metric label in aggregation context",
			query:           `sum(metric_name_one{`,
			expectedMatches: sets.NewString("dima", "dimb"),
		},
		{
			desc:            "completes on aggregation keywords",
			query:           `sum(metric_name_one)`,
			expectedMatches: sets.StringKeySet(aggregateKeywords),
		},
		{
			desc:            "completes on metric name",
			query:           `metric_name`,
			expectedMatches: sets.NewString("metric_name_one", "metric_name_two"),
		},
		{
			desc:            "completes on empty string",
			query:           ``,
			expectedMatches: sets.NewString("metric_name_one", "metric_name_two").Union(sets.StringKeySet(aggregators)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			c := NewPromQLCompleter(index)
			matches := c.GenerateSuggestions(tc.query, len(tc.query))
			matchVals := toSet(matches)
			if !reflect.DeepEqual(matchVals, tc.expectedMatches) {
				t.Errorf("\n\nGot %v matches [%v], expected %v\n\n", len(matchVals), matchVals, tc.expectedMatches)
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
