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

package prom

import (
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"

	"k8s.io/instrumentation-tools/notstdlib/sets"
)

func TestIndexUpdatesAgainstMetricNames(t *testing.T) {
	index := NewTestIndex()
	now := time.Now()
	testCases := []struct {
		name                string
		scrapeMetricsString *string
		want                sets.String
	}{
		{
			name:                "should be empty initially",
			scrapeMetricsString: nil,
			want:                sets.String{},
		},
		{
			name: "should be able to update",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total 1
`),
			want: sets.NewString("han_metric_total"),
		},
		{
			name: "should be able to update with existing metric",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total 1
# HELP han_metric_total2 [STABLE] counter help
# TYPE han_metric_total2 counter
han_metric_total2 2
`),
			want: sets.NewString("han_metric_total", "han_metric_total2"),
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now = now.Add(time.Duration(i) * time.Second)
			if tc.scrapeMetricsString != nil {
				err := index.LoadMetrics(*tc.scrapeMetricsString, now)
				if err != nil {
					t.Errorf("didn't expect this to err %v", err)
				}
			}
			if got := index.GetMetricNames(); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("GetMetricNames() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIndexUpdatesAgainstMetricDimensions(t *testing.T) {
	index := NewTestIndex()
	now := time.Now()
	primaryMetric := "han_metric_total"
	secondaryMetric := "other_metric_name"
	testCases := []struct {
		name                string
		scrapeMetricsString *string
		wantPrimary         sets.String
		wantSecondary       sets.String
	}{
		{
			name:                "should be empty initially",
			scrapeMetricsString: nil,
			wantPrimary:         sets.String{},
			wantSecondary:       sets.String{},
		},
		{
			name: "should be able to update",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{d="1"} 1
`),
			wantPrimary:   sets.NewString("d"),
			wantSecondary: sets.String{},
		},
		{
			name: "more values for a existing dimension should not affect the number of dimensions we have",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{d="1"} 2
han_metric_total{d="2"} 1
`),
			wantPrimary:   sets.NewString("d"),
			wantSecondary: sets.String{},
		},
		{
			name: "new dimensions should affect the number of dimensions we have",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{d="1"} 2
han_metric_total{d="2"} 1
han_metric_total{d="1", other="2"} 3
`),
			wantPrimary:   sets.NewString("d", "other"),
			wantSecondary: sets.String{},
		},
		{
			name: "if we add another metric, we should also store that metric's dimension data",
			scrapeMetricsString: proto.String(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{d="1"} 2
han_metric_total{d="2"} 1
han_metric_total{d="1", other="2"} 3
# HELP other_metric_name [STABLE] counter help
# TYPE other_metric_name counter
other_metric_name{blah="a"} 2
`),
			wantPrimary:   sets.NewString("d", "other"),
			wantSecondary: sets.NewString("blah"),
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now = now.Add(time.Duration(i) * time.Second)
			if tc.scrapeMetricsString != nil {
				err := index.LoadMetrics(*tc.scrapeMetricsString, now)
				if err != nil {
					t.Errorf("didn't expect this to err %v", err)
				}
			}
			if got := index.GetStoredDimensionsForMetric(primaryMetric); !reflect.DeepEqual(got, tc.wantPrimary) {
				t.Errorf("GetStoredDimensionsForMetric(%v) = %v, want %v", primaryMetric, got, tc.wantPrimary)
			}
			if got := index.GetStoredDimensionsForMetric(secondaryMetric); !reflect.DeepEqual(got, tc.wantSecondary) {
				t.Errorf("GetStoredDimensionsForMetric(%v) = %v, want %v", secondaryMetric, got, tc.wantSecondary)
			}
		})
	}
}

func TestIndexUpdatesAgainstMetricDimensionValues(t *testing.T) {
	index := NewIndex()
	now := time.Now()
	primaryMetric := "han_metric_total"
	secondaryMetric := "other_metric_name"
	testCases := []struct {
		name                string
		scrapeMetricsString []byte
		wantPrimaryKeyone   sets.String
		wantPrimaryKeytwo   sets.String
		wantSecondaryKeyone sets.String
	}{
		{
			name:                "should be empty initially",
			scrapeMetricsString: nil,
			wantPrimaryKeyone:   nil,
			wantPrimaryKeytwo:   nil,
			wantSecondaryKeyone: nil,
		},
		{
			name: "should be able to update",
			scrapeMetricsString: []byte(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{keyone="a"} 1
`),
			wantPrimaryKeyone:   sets.NewString("a"),
			wantPrimaryKeytwo:   nil,
			wantSecondaryKeyone: nil,
		},
		{
			name: "more values for a existing dimension should not affect the number of dimensions we have",
			scrapeMetricsString: []byte(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{keyone="a"} 2
han_metric_total{keyone="b"} 1
`),
			wantPrimaryKeyone:   sets.NewString("a", "b"),
			wantPrimaryKeytwo:   nil,
			wantSecondaryKeyone: nil,
		},
		{
			name: "new dimensions should affect the number of dimensions we have",
			scrapeMetricsString: []byte(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{keyone="a"} 2
han_metric_total{keyone="b"} 1
han_metric_total{keyone="c", keytwo="d"} 3
`),
			wantPrimaryKeyone:   sets.NewString("a", "b", "c"),
			wantPrimaryKeytwo:   sets.NewString("d"),
			wantSecondaryKeyone: nil,
		},
		{
			name: "if we add another metric, we should also store that metric's dimension data",
			scrapeMetricsString: []byte(`
# HELP han_metric_total [STABLE] counter help
# TYPE han_metric_total counter
han_metric_total{keyone="a"} 2
han_metric_total{keyone="b"} 1
han_metric_total{keyone="c", keytwo="d"} 3
# HELP other_metric_name [STABLE] counter help
# TYPE other_metric_name counter
other_metric_name{keyone="a"} 2
`),
			wantPrimaryKeyone:   sets.NewString("a", "b", "c"),
			wantPrimaryKeytwo:   sets.NewString("d"),
			wantSecondaryKeyone: sets.NewString("a"),
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			now = now.Add(time.Duration(i) * time.Second)
			if tc.scrapeMetricsString != nil {
				pms, err := ParseTextData(tc.scrapeMetricsString, now)
				if err != nil {
					t.Errorf("didn't expect this to err %v", err)
				}
				for _, m := range pms {
					index.UpdateMetric(m)
				}
			}
			if got := index.GetStoredValuesForMetricAndDimension(primaryMetric, "keyone"); !reflect.DeepEqual(got, tc.wantPrimaryKeyone) {
				t.Errorf("\nGetStoredValuesForMetricAndDimension(%v, keyone) = %v, want %v\n", primaryMetric, got, tc.wantPrimaryKeyone)
			}
			if got := index.GetStoredValuesForMetricAndDimension(primaryMetric, "keytwo"); !reflect.DeepEqual(got, tc.wantPrimaryKeytwo) {
				t.Errorf("\nGetStoredValuesForMetricAndDimension(%v, keytwo) = %v, want %v\n", primaryMetric, got, tc.wantPrimaryKeytwo)
			}
			if got := index.GetStoredValuesForMetricAndDimension(secondaryMetric, "keyone"); !reflect.DeepEqual(got, tc.wantSecondaryKeyone) {
				t.Errorf("\nGetStoredValuesForMetricAndDimension(%v, keyone) = %v, want %v\n", primaryMetric, got, tc.wantSecondaryKeyone)
			}
		})
	}
}

type TestIndex struct {
	Indexer
}

func (ti *TestIndex) LoadMetrics(rawMetricsText string, time time.Time) error {
	parsedSeries, err := ParseTextData([]byte(rawMetricsText), time)
	if err != nil {
		return err
	}
	for _, series := range parsedSeries {
		ti.UpdateMetric(series)
	}
	return nil
}

func NewTestIndex() *TestIndex {
	return &TestIndex{NewIndex()}
}

func NewTestIndexFromData(rawMetricsText string, time time.Time) (*TestIndex, error) {
	index := NewTestIndex()
	err := index.LoadMetrics(rawMetricsText, time)
	if err != nil {
		return nil, err
	}
	return index, nil
}
