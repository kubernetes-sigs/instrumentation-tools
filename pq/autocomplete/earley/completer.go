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
	"sort"
	"strings"

	"k8s.io/instrumentation-tools/debug"
	"k8s.io/instrumentation-tools/notstdlib/sets"
	"k8s.io/instrumentation-tools/pq/autocomplete"
)

const (
	PromQLTokenSeparators = " []{}()=!~,"
)

type matchResult struct {
	Value  string // this is the text for completion
	Kind   string // type of match from which this result is populated
	Detail string // additional information that may be displayed for auto-complete
}

func (m matchResult) GetValue() string {
	return m.Value
}

func (m matchResult) GetKind() string {
	return m.Kind
}

func (m matchResult) GetDetail() string {
	return m.Detail
}

func NewPartialMatch(name, kind, detail string) autocomplete.Match {
	return &matchResult{Value: name, Kind: kind, Detail: detail}
}

func NewPromQLCompleter(index autocomplete.QueryIndex) autocomplete.PromQLCompleter {
	return &promQLCompleter{
		index: index,
	}
}

type promQLCompleter struct {
	autocomplete.PromQLCompleter
	index autocomplete.QueryIndex
}

func (c *promQLCompleter) GetMetricNames() sets.String {
	return c.index.GetMetricNames()
}

func (c *promQLCompleter) GetStoredDimensionsForMetric(mName string) sets.String {
	return c.index.GetStoredDimensionsForMetric(mName)
}

func (c *promQLCompleter) GetStoredValuesForMetricAndDimension(mName, lName string) sets.String {
	return autocomplete.Enquote(c.index.GetStoredValuesForMetricAndDimension(mName, lName))
}

func (c *promQLCompleter) SuggestParens(query string, pos int, isPrecededByWhiteSpace bool) sets.String {
	if isPrecededByWhiteSpace {
		return sets.NewString("(")
	}
	return sets.String{}
}

// GenerateSuggestions has the glue code for taking our token types and mapping
// them to a concrete list of suggestion via our indexer. We compute our autocomplete
// prefix (i.e. the incomplete text at the cursor position) and use that to filter
// against our concrete list.
func (c *promQLCompleter) GenerateSuggestions(query string, pos int) []autocomplete.Match {
	var matches []autocomplete.Match
	q := query[0:pos]
	autocompletePrefix := getPrefix(q)
	debug.Debugf("\n\nautocomplete prefix: '%v'\n\n", autocompletePrefix)

	q = q[0 : len(q)-len(autocompletePrefix)]
	tokens := extractWords(q)
	suggestions := PromQLParser.GetSuggestedTokenType(tokens)

	for _, s := range suggestions {
		switch s.TokenType {
		case METRIC_LABEL_SUBTYPE:
			if s.ctx.HasMetric() {
				metricName := s.ctx.GetMetric()
				for _, d := range autocomplete.FilterPrefix(c.GetStoredDimensionsForMetric(metricName), autocompletePrefix, false).List() {
					values := c.GetStoredValuesForMetricAndDimension(metricName, d).List()
					newMatch := NewPartialMatch(d, "metric-label", strings.Join(values, ","))
					matches = append(matches, newMatch)
				}
			}
		case METRIC_ID:
			metricMatches := autocomplete.FilterPrefix(c.GetMetricNames(), autocompletePrefix, false)
			for _, m := range metricMatches.List() {
				dims := c.GetStoredDimensionsForMetric(m).List()
				newMatch := NewPartialMatch(m, "metric-id", strings.Join(dims, ","))
				matches = append(matches, newMatch)
			}
		case STRING:
			if s.ctx.HasMetric() && s.ctx.HasMetricLabel() {
				for _, m := range autocomplete.FilterPrefix(c.GetStoredValuesForMetricAndDimension(s.ctx.GetMetric(), s.ctx.GetMetricLabel()), autocompletePrefix, false).List() {
					dims := c.GetStoredDimensionsForMetric(m).List()
					newMatch := NewPartialMatch(m, "metric-id", strings.Join(dims, ","))
					matches = append(matches, newMatch)
				}
			}
		case AGGR_OP:
			for _, ao := range autocomplete.FilterPrefix(sets.StringKeySet(aggregators), autocompletePrefix, false).List() {
				newMatch := NewPartialMatch(ao, "aggr-operation", aggregators[ao])
				matches = append(matches, newMatch)
			}
		case AGGR_KW:
			for _, ao := range autocomplete.FilterPrefix(sets.StringKeySet(aggregateKeywords), autocompletePrefix, false).List() {
				newMatch := NewPartialMatch(ao, "aggr-keyword", aggregateKeywords[ao])
				matches = append(matches, newMatch)
			}
		}
		sort.Slice(matches, func(i, j int) bool {
			return matches[i].GetValue() > matches[j].GetValue()
		})
	}
	return matches
}

func getPrefix(query string) string {
	if len(query) == 0 {
		return ""
	}
	for i := len(query) - 1; i >= 0; i-- {
		c := []rune(query)[i]
		if strings.ContainsRune(PromQLTokenSeparators, c) {
			return query[i+1:]
		}
	}
	return query
}
