package charma

import "strings"

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

func CircleCode(input string) string {
	result := []rune{}

	for _, r := range input {
		if circle, found := circleChars[r]; found {
			result = append(result, circle)
		} else {
			result = append(result, r)
		}
	}

	return strings.Join(strings.Split(string(result), ""), " ")
}
