/*
Copyright 2020 The Kubernetes Authors.

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

package prom

import (
	"context"
	"fmt"
	"github.com/prometheus/prometheus/promql/parser"
	"sync"
	"time"

	"github.com/prometheus/prometheus/promql"
)

// ResultsCallback is a function that processes the results of a prometheus query
// The given results are *only* valid for the life of the query, and must be deep-copied
// if they are kept around.
type ResultsCallback func(*promql.Result) error

type DataSource interface {
	ScrapePrometheusEndpoint(ctx context.Context, nowish time.Time) ([]ParsedSeries, error)
}

type Range struct {
	Window   time.Duration
	Interval time.Duration
	Instant  bool
}

type PeriodicData struct {
	source DataSource

	storageMu sync.RWMutex
	storage   *rangeStorage
	engine    *promql.Engine
	queryMu   sync.RWMutex
	Callback  ResultsCallback
	Query     string
	Times     Range
	index    Indexer
}

func NewPeriodicData(source DataSource, opts promql.EngineOpts) *PeriodicData {
	return &PeriodicData{
		source:  source,
		storage: NewRangeStorage(),
		engine:  promql.NewEngine(opts),
		index:  NewIndex(),
	}
}

func (q *PeriodicData) SetQuery(ctx context.Context, query string) error {
	q.queryMu.Lock()
	defer q.queryMu.Unlock()
	// let's validate that the querystring is parseable
	_, err := parser.ParseExpr(query)
	if err != nil {
		return err
	}
	q.Query = query
	return nil
}

func (q *PeriodicData) Scrape(ctx context.Context) error {
	q.storageMu.Lock()
	defer q.storageMu.Unlock()
	data, err := q.source.ScrapePrometheusEndpoint(ctx, time.Now())
	if err != nil {
		return fmt.Errorf("unable to get new data from source: %w", err)
	}
	for _, d := range data {
		q.index.UpdateMetric(d)
	}
	if err := func() error {
		if err := q.storage.LoadData(data); err != nil {
			return fmt.Errorf("unable to load new data, may now be in inconsistent state: %w", err)
		}
		// release the defer before we send the notification
		return nil
	}(); err != nil {
		return err
	}

	if err := q.ManuallyExecuteQuery(ctx, q.Callback); err != nil {
		return fmt.Errorf("unable to execute query: %w", err)
	}
	return nil
}

func (q *PeriodicData) ManuallyExecuteQuery(ctx context.Context, cb ResultsCallback) error {
	var query promql.Query
	if q.Times.Instant {
		var err error
		query, err = q.engine.NewInstantQuery(q.storage, q.Query, time.Now())
		if err != nil {
			return fmt.Errorf("unable to construct instant query: %w", err)
		}
	} else {
		var err error
		end := time.Now()
		start := time.Now().Add(time.Duration(-1) * q.Times.Window)
		query, err = q.engine.NewRangeQuery(q.storage, q.Query, start, end, q.Times.Interval)
		if err != nil {
			return fmt.Errorf("unable to construct range query: %w", err)
		}
	}
	defer query.Close()
	// NB(directxman12): THE QUERY DATA IS ONLY VALID INSIDE THIS FUNCTION
	return cb(query.Exec(ctx))
}

func (q *PeriodicData) GetIndex() Indexer {
	return q.index
}

func DefaultEngineOptions(timeout time.Duration, maxSamples int) promql.EngineOpts {
	// TODO(sollyross): add logging
	// TODO(sollyross): figure out good options
	return promql.EngineOpts{
		Timeout:    timeout, // why? why is this here? THIS IS WHAT CONTEXT IS FOR!
		MaxSamples: maxSamples,
		// TODO(sollyross): anything else
	}
}
