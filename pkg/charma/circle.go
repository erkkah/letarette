// Copyright 2019 Erik Agsjö
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package charma

var circleChars = map[rune]rune{}

func init() {
	numChars := 'Z' - 'A'
	for i := rune(0); i < numChars; i++ {
		capital := 'A' + i
		small := 'a' + i

		capitalCircle := i + rune(0x24b6)
		circleChars[capital] = capitalCircle

		smallCircle := i + rune(0x24d0)
		circleChars[small] = smallCircle
	}
}

// CircleChars replaces all A-Z a-z characters in
// the provided string with the circled variants.
func CircleChars(input string) string {
	result := []rune{}

	lastChar := len(input)
	currentChar := 0
	for _, r := range input {
		currentChar++
		if circle, found := circleChars[r]; found {
			result = append(result, circle)
			if currentChar != lastChar {
				result = append(result, ' ')
			}
		} else {
			result = append(result, r)
		}
	}

	return string(result)
}
