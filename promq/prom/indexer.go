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
	"sync"

	"github.com/prometheus/prometheus/pkg/labels"

	debug "sigs.k8s.io/instrumentation-tools/debug/error"
	"sigs.k8s.io/instrumentation-tools/notstdlib/sets"
)

type Indexer interface {
	UpdateMetric(m ParsedSeries)
	GetMetricNames() sets.String
	GetStoredDimensionsForMetric(string) sets.String
	GetStoredValuesForMetricAndDimension(string, string) sets.String
}

type indexer struct {
	metricNameMu sync.RWMutex
	// let's just be super inefficient
	store map[string]map[string]sets.String
	// metric bloom filter
	metricBloomFilter sets.Uint64
}

func NewIndex() Indexer {
	return &indexer{
		metricNameMu:      sync.RWMutex{},
		metricBloomFilter: sets.Uint64{},
		store:             map[string]map[string]sets.String{},
	}
}

func (i *indexer) isMetricPresent(hash uint64) bool {
	// check our metricBloomFilter and abort if unnecessary
	i.metricNameMu.RLock()
	defer i.metricNameMu.RUnlock()
	return i.metricBloomFilter.Has(hash)
}

func (i *indexer) UpdateMetric(m ParsedSeries) {
	hash := m.Labels.Hash()
	// abort if we don't need to store this parsed series.
	// note: we don't care about collisions, this is functionally
	// a bloom filter.
	if i.isMetricPresent(hash) {
		return
	}
	ls := m.Labels.Map()
	n, ok := ls[labels.MetricName]
	if !ok {
		debug.Errorln("This metric doesn't have a name")
		return
	}
	i.metricNameMu.Lock()
	defer i.metricNameMu.Unlock()
	// next time we will know that
	i.metricBloomFilter.Insert(hash)
	if _, ok := i.store[n]; !ok {
		i.store[n] = map[string]sets.String{}
	}

	for l, v := range ls {
		if l == labels.MetricName {
			continue
		}
		if _, ok := i.store[n][l]; !ok {
			i.store[n][l] = sets.NewString()
		}
		i.store[n][l].Insert(v)
	}
}

func (i *indexer) GetMetricNames() sets.String {
	i.metricNameMu.RLock()
	defer i.metricNameMu.RUnlock()
	return sets.StringKeySet(i.store)
}

func (i *indexer) GetStoredDimensionsForMetric(metricName string) sets.String {
	i.metricNameMu.RLock()
	defer i.metricNameMu.RUnlock()
	return sets.StringKeySet(i.store[metricName])

}

func (i *indexer) GetStoredValuesForMetricAndDimension(metricName string, dimension string) sets.String {
	i.metricNameMu.RLock()
	defer i.metricNameMu.RUnlock()
	dimensionForMetric, ok := i.store[metricName]
	if !ok {
		return nil
	}
	return dimensionForMetric[dimension]
}
