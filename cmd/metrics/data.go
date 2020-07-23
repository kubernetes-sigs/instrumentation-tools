/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package metrics

import (
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/promql"

	"sigs.k8s.io/instrumentation-tools/promq/term/plot"
)

// TODO(sollyross): can we make this more efficient with prometheus return data?  do we need to?
type PromSeriesSet promql.Matrix

func PromResultToPromSeriesSet(res *promql.Result) (plot.SeriesSet, error) {
	if res.Err != nil {
		return nil, res.Err
	}
	rawSeriesSet, err := res.Matrix()
	if err != nil {
		return nil, fmt.Errorf("data was not a Prometheus Matrix: %w", err)
	}

	set := make(plot.SeriesSet, len(rawSeriesSet))
	for i, series := range rawSeriesSet {
		set[i] = &PromSeries{Series: series}
	}

	return set, nil
}

type PromSeries struct {
	promql.Series
}

func (s *PromSeries) Title() string {
	sb := &strings.Builder{}
	for i, lbl := range s.Metric {
		if i != 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(lbl.Value)
	}
	return sb.String()
}
func (s *PromSeries) Id() plot.SeriesId {
	// TODO: this is stable, but not guaranteed to be unique
	return plot.SeriesId(s.Metric.Hash() % 255 + 1)
}

func (s *PromSeries) Points() []plot.Point {
	res := make([]plot.Point, len(s.Series.Points))
	for i, pt := range s.Series.Points {
		res[i] = PromPoint(pt)
	}
	return res
}

type PromPoint promql.Point

func (p PromPoint) X() int64 {
	return p.T
}
func (p PromPoint) Y() float64 {
	return p.V
}
