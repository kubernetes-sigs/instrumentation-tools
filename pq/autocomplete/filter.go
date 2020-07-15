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

import (
	"strconv"
	"strings"

	"k8s.io/instrumentation-tools/notstdlib/sets"
)

// a lot of this is lifted from "github.com/c-bata/go-prompt" but typed against sets.String
// since they are helpful in pruning a list of suggestions, given the filtering constraints.
// FilterPrefix takes a set of strings and compares each string against the prefix and includes
// the string if the string starts with that prefix.
func FilterPrefix(stringSet sets.String, prefix string, ignoreCase bool) sets.String {
	if prefix == "" {
		return stringSet
	}
	return filterSet(stringSet, prefix, ignoreCase, strings.HasPrefix)
}

// FilterFuzzy takes a set of strings and compares each string against the prefix and includes
// the string if the string contains the prefix in a subsequence, i.e. fuzzy searching for 'dog
// is equivalent to "*d*o*g*", which matches "Good food is gone".
func FilterFuzzy(stringSet sets.String, prefix string, ignoreCase bool) sets.String {
	if prefix == "" {
		return stringSet
	}
	return filterSet(stringSet, prefix, ignoreCase, _fuzzyMatch)
}

// filterSet takes a set of strings (your starting strings), a string representing your
// desired match (some regex), ignoreCase for whether you want to ignore case, and a func which
// stores a comparison func for your string.
func filterSet(autocompletions sets.String, sub string, ignoreCase bool, inclusionFunc func(string, string) bool) sets.String {
	if sub == "" {
		return autocompletions
	}
	if ignoreCase {
		sub = strings.ToLower(sub)
	}
	ret := sets.NewString()
	for _, item := range autocompletions.List() {
		if ignoreCase {
			item = strings.ToLower(item)
		}
		if inclusionFunc(item, sub) {
			ret.Insert(item)
		}
	}
	return ret
}

func Enquote(stringSet sets.String) sets.String {
	newStrings := sets.NewString()
	for _, item := range stringSet.List() {
		newStrings.Insert(strconv.Quote(item))
	}
	return newStrings
}

func _fuzzyMatch(s, sub string) bool {
	sChars := []rune(s)
	subChars := []rune(sub)
	sIdx := 0

	for _, c := range subChars {
		found := false
		for ; sIdx < len(sChars); sIdx++ {
			if sChars[sIdx] == c {
				found = true
				sIdx++
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
