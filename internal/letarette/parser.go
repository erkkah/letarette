package letarette

import (
	"bytes"
	"text/scanner"
	"unicode"

	"github.com/erkkah/letarette/pkg/logger"
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
		if r == '*' {
			return false
		}
		return unicode.IsGraphic(r) && !unicode.IsSpace(r)
	}

	var result []Phrase
	excludeNext := false

	for tok := s.Scan(); tok != scanner.EOF; tok = s.Scan() {
		switch tok {
		case scanner.String:
			fallthrough
		case scanner.Ident:
			result = append(result, Phrase{
				Text:    s.TokenText(),
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
			logger.Debug.Printf("Unexpected parse result: %c", tok)
		}
	}

	return result
}
