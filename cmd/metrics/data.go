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

// TODO: this practically has no reason to be an interface any more, and we can
// probably save space by making it not one

// TODO(sollyross): can we make this more efficient with prometheus return data?  do we need to?
type PromSeriesSet promql.Matrix

// PromResultToPromSeriesSet copies res and converts it to a format suitable
// for use with the terminal plotting library.  It copies since query results
// are generally not usable beyond the lifetime of the query.
func PromResultToPromSeriesSet(res *promql.Result) (plot.SeriesSet, error) {
	if res.Err != nil {
		return nil, res.Err
	}
	rawSeriesSet, err := res.Matrix()
	if err != nil {
		return nil, fmt.Errorf("data was not a Prometheus Matrix: %w", err)
	}

	set := make(plot.SeriesSet, len(rawSeriesSet))
	for i, origSeries := range rawSeriesSet {
		var title string
		{
			sb := &strings.Builder{}
			for i, lbl := range origSeries.Metric {
				if i != 0 {
					sb.WriteString(", ")
				}
				sb.WriteString(lbl.Value)
			}
			title = sb.String()
		}

		// TODO: this is stable, but not guaranteed to be unique
		id :=  plot.SeriesId(origSeries.Metric.Hash() % 255 + 1)

		series := &PromSeries{
			title: title,
			id: id,
		}

		series.points = make([]plot.Point, len(origSeries.Points))
		for i, point := range origSeries.Points {
			series.points[i] = PromPoint(point)
		}

		set[i] = series
	}

	return set, nil
}

type PromSeries struct {
	title string
	id plot.SeriesId
	points []plot.Point
}

func (s *PromSeries) Title() string {
	return s.title
}
func (s *PromSeries) Id() plot.SeriesId {
	return s.id
}

func (s *PromSeries) Points() []plot.Point {
	return s.points
}

type PromPoint promql.Point

func (p PromPoint) X() int64 {
	return p.T
}
func (p PromPoint) Y() float64 {
	return p.V
}
