// Package kvargs has functions of transforming maps into key value args
package kvargs

import (
	"sort"
	"strings"
)

// MapToSortedArgs {b: "2", a: "1"} -> [argFn("a", "1")..., argFn("b", "2")...]
func MapToSortedArgs(m map[string]string, argFn func(k, v string) []string) []string {
	sortedKeys := make([]string, 0, len(m))
	for k := range m {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)
	s := make([]string, 0, len(m)*2)
	for _, k := range sortedKeys {
		s = append(s, argFn(k, m[k])...)
	}
	return s
}

// OptionArg build a arg build function
// {"k": "v", ...} ->  ["-k"+suffix, v, ...]
func OptionArg(suffix string) func(k, v string) []string {
	return func(opt, value string) []string {
		if !strings.HasPrefix(opt, "-") {
			opt = "-" + opt
		}
		return []string{opt + suffix, value}
	}
}
