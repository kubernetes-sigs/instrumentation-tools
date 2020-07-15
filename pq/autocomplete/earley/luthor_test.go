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
)

func TestExtractWords(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantWords []string
	}{
		{
			name:      "Should only have EOF",
			input:     "",
			wantWords: []string{""},
		},
		{
			name:      "Should have the same head and tail token when we have only a single token",
			input:     "start",
			wantWords: []string{"start", ""},
		},
		{
			name:      "Should have the different head and tail when we have 1+ tokens",
			input:     "start{blah='aaa'}",
			wantWords: []string{"start", "{", "blah", "=", "'aaa'", "}", ""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractWords(tt.input); !reflect.DeepEqual(got.Vals(), tt.wantWords) {
				t.Errorf("extractWords() = %v, want %v", got.Vals(), tt.wantWords)
			}
		})
	}
}
