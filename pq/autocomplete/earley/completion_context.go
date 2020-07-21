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
	"github.com/golang/protobuf/proto"

	"sigs.k8s.io/instrumentation-tools/notstdlib/sets"
)

type ContextualToken struct {
	TokenType
	ctx *completionContext
}

type CompletionContext interface {
	HasMetric() bool
	GetMetric() string
	HasMetricLabel() bool
	GetMetricLabel() string
	GetUsedMetricLabelValues() sets.String
}

type completionContext struct {
	metric            *string
	metricLabel       *string
	metricLabelValues sets.String
}

// Build context with input token
func (c *completionContext) BuildContext(tokenType *TokenType, token *Tokhan) {
	switch *tokenType {
	case METRIC_ID:
		c.metric = proto.String(token.Val)
	case METRIC_LABEL_SUBTYPE:
		c.metricLabel = proto.String(token.Val)
	case STRING:
		c.AddObservedMetricLabelValue(token.Val)
	default:
	}
}

func (c *completionContext) HasMetric() bool {
	return c.metric != nil
}

func (c *completionContext) AddObservedMetricLabelValue(labelVal string) {
	lv := sets.NewString(labelVal)
	c.metricLabelValues = lv.Union(c.metricLabelValues)
}

func (c *completionContext) GetMetric() string {
	return *c.metric
}

func (c *completionContext) HasMetricLabel() bool {
	return c.metricLabel != nil
}

func (c *completionContext) GetMetricLabel() string {
	return *c.metricLabel
}

func (c *completionContext) GetUsedMetricLabelValues() sets.String {
	if c.metricLabelValues == nil {
		return sets.NewString()
	}
	return c.metricLabelValues
}
