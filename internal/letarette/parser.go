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

package letarette

/*

Search syntax:

<phrase> ::= string | quotedstring
<query> ::= [-] <phrase> [*]
<query> ::= <query> <query>

Where the '-' prefix means "not" and the '*' denotes wildcard searches.

Examples:

animal -dog -cat

horse* -"horse head"

The output of the search parser is a list of including phrases and a list of
excluding phrases. Both lists can contain wildcard expressions, which will lead
to prefix searches.

The parser is very defensive and will always produce a valid query.

Searches will always be performed as "near" queries for all including phrases
followed by a NOT list built from all excluding phrases.

*/

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"text/scanner"
	"unicode"
)

// Phrase represents one parsed query phrase, with flags
type Phrase struct {
	Text     string
	Wildcard bool
	Exclude  bool
}

func (p Phrase) String() string {
	prefix := ""
	if p.Exclude {
		prefix = "-"
	}
	suffix := ""
	if p.Wildcard {
		suffix = "*"
	}
	phraseText := p.Text
	if strings.Contains(phraseText, " ") && !strings.HasPrefix(phraseText, `"`) {
		phraseText = fmt.Sprintf("%q", phraseText)
	}
	return prefix + phraseText + suffix
}

// ParseQuery tokenizes a query string and returns a list
// of parsed phrases with exclusion and wildcard flags.
func ParseQuery(query string) []Phrase {
	var s scanner.Scanner
	s.Init(bytes.NewBufferString(query))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings
	s.IsIdentRune = func(r rune, i int) bool {
		if r == '-' && i == 0 {
			return false
		}
		if r == '*' || r == '"' || r == '\'' || r == '(' || r == ')' {
			return false
		}
		return unicode.IsGraphic(r) && !unicode.IsSpace(r)
	}

	var result []Phrase
	excludeNext := false

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		text := s.TokenText()
		switch tok {
		case scanner.Ident:
			//text = fmt.Sprintf("%q", text)
			fallthrough
		case scanner.String:
			text = unquote(text)
			result = append(result, Phrase{
				Text:    text,
				Exclude: excludeNext,
			})
			excludeNext = false
		case '-':
			excludeNext = true
		case '*':
			l := len(result)
			if l > 0 {
				result[l-1].Wildcard = true
			}
		default:
			// skip
		}
	}

	return result
}

var singleChars = regexp.MustCompile(`\PL\pL\PL`)
var singleCharStart = regexp.MustCompile(`^\pL\PL`)
var singleCharEnd = regexp.MustCompile(`\PL\pL$`)
var whiteSpaces = regexp.MustCompile(`\s+`)

func unquote(phrase string) string {
	return strings.TrimSuffix(strings.TrimPrefix(phrase, `"`), `"`)
}

func reducePhrase(phrase string) string {
	reduced := unquote(phrase)

	// Cut single character phrase at once
	if len(reduced) == 1 && !unicode.IsNumber([]rune(reduced)[0]) {
		return ""
	}
	reduced = singleChars.ReplaceAllString(reduced, " ")
	reduced = singleCharStart.ReplaceAllString(reduced, " ")
	reduced = singleCharEnd.ReplaceAllString(reduced, " ")
	reduced = whiteSpaces.ReplaceAllString(reduced, " ")
	reduced = strings.TrimSpace(reduced)
	if reduced == " " {
		return ""
	}
	return reduced
}

// ReducePhraseList removes one character phrases
// from a list of phrases.
func ReducePhraseList(phrases []Phrase) []Phrase {
	var result []Phrase
	for _, phrase := range phrases {
		phrase.Text = reducePhrase(phrase.Text)
		if len(phrase.Text) > 0 {
			result = append(result, phrase)
		}
	}
	return result
}

// CanonicalizePhraseList turns all phrases in a phrase list to lower case,
// sorts it and eliminates duplicates.
func CanonicalizePhraseList(phrases []Phrase) []Phrase {
	result := append(phrases[:0:0], phrases...)
	if len(result) == 0 {
		return result
	}
	for i, v := range result {
		result[i].Text = strings.ToLower(v.Text)
	}
	sort.Slice(result, func(i int, j int) bool {
		strDiff := strings.Compare(result[i].Text, result[j].Text)
		if strDiff == -1 {
			return true
		} else if strDiff == 1 {
			return false
		}
		if result[i].Exclude != result[j].Exclude {
			return result[j].Exclude
		}
		if result[i].Wildcard != result[j].Exclude {
			return result[j].Wildcard
		}
		return false
	})

	last := result[0]
	unique := []Phrase{last}
	for _, v := range result[1:] {
		if v != last {
			unique = append(unique, v)
		}
		last = v
	}
	return unique
}
