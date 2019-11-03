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
	return prefix + p.Text + suffix
}

// ParseQuery tokenizes a query string and returns a list
// of parsed phrases with exclusion and wildcard flags.
func ParseQuery(query string) []Phrase {
	var s scanner.Scanner
	s.Init(bytes.NewBufferString(query))
	s.Mode = scanner.ScanIdents | scanner.ScanStrings
	s.IsIdentRune = func(r rune, i int) bool {
		if r == '"' && i == 0 {
			return false
		}
		if r == '-' && i == 0 {
			return false
		}
		if r == '*' || r == '\'' || r == '(' || r == ')' {
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
			text = fmt.Sprintf("%q", text)
			fallthrough
		case scanner.String:
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
