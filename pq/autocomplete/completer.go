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

package autocomplete

import "k8s.io/instrumentation-tools/notstdlib/sets"

// in order to generate completion results, we require some store
// to implement an interface for retrieving metric names, their
// associated label keys and also their label values, since these
// values cannot be hardcoded but must be inferred at runtime.
type QueryIndex interface {
	GetMetricNames() sets.String
	GetStoredDimensionsForMetric(string) sets.String
	GetStoredValuesForMetricAndDimension(string, string) sets.String
}
type Match interface {
	GetValue() string
	GetKind() string
	GetDetail() string
}
type PromQLCompleter interface {
	QueryIndex
	GenerateSuggestions(query string, pos int) []Match
}
