/*
Copyright 2021 gRPC authors.

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

package xunit

import (
	"strings"
	"unicode"
)

// Dashify returns the input string where all whitespace and underscore
// characters have been replaced by dashes and, aside from dashes, only
// alphanumeric characters remain. This is designed to be used to strip the
// "id" XML attribute of special characters.
func Dashify(s string) string {
	b := strings.Builder{}
	for _, rune := range s {
		if string(rune) == "_" || unicode.IsSpace(rune) {
			b.WriteString("-")
		} else if string(rune) == "-" || unicode.IsLetter(rune) || unicode.IsNumber(rune) {
			b.WriteRune(rune)
		}
	}
	return b.String()
}
