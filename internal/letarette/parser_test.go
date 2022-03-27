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

package letarette_test

import (
	"fmt"
	"testing"

	"github.com/erkkah/letarette/internal/letarette"
	xt "github.com/erkkah/letarette/pkg/xt"
)

func TestParsePlainPhrases(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery("cat dog banana")

	xt.Assert(len(r) == 3)

	xt.Assert(r[0] == letarette.Phrase{
		`cat`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`dog`, false, false,
	})

	xt.Assert(r[2] == letarette.Phrase{
		`banana`, false, false,
	})
}

func TestIncludeExcludePhrases(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery("cat -dog banana - fishtank")
	xt.Assert(len(r) == 4)

	xt.Assert(r[0] == letarette.Phrase{
		`cat`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`dog`, false, true,
	})

	xt.Assert(r[2] == letarette.Phrase{
		`banana`, false, false,
	})

	xt.Assert(r[3] == letarette.Phrase{
		`fishtank`, false, true,
	})
}

func TestWildcardPhrases(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery("cat* -dog * banana - fishtank")
	xt.Assert(len(r) == 4)

	xt.Assert(r[0] == letarette.Phrase{
		`cat`, true, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`dog`, true, true,
	})

	xt.Assert(r[2] == letarette.Phrase{
		`banana`, false, false,
	})

	xt.Assert(r[3] == letarette.Phrase{
		`fishtank`, false, true,
	})
}

func TestEmbeddedAndFreeExcludes(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery("cat- cat-litter - -dog -")
	xt.Assert(len(r) == 3)

	xt.Assert(r[0] == letarette.Phrase{
		`cat-`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`cat-litter`, false, false,
	})

	xt.Assert(r[2] == letarette.Phrase{
		`dog`, false, true,
	})
}

func TestEmbeddedWildcard(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`cat* cat*litter "*dog*"`)
	xt.Assert(len(r) == 4)

	xt.Assert(r[0] == letarette.Phrase{
		`cat`, true, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`cat`, true, false,
	})

	xt.Assert(r[2] == letarette.Phrase{
		`litter`, false, false,
	})

	xt.Assert(r[3] == letarette.Phrase{
		`*dog*`, false, false,
	})
}

func TestString(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`"cat - * - dog" "kawo\"nka"*`)
	xt.Assert(len(r) == 2)

	xt.Assert(r[0] == letarette.Phrase{
		`cat - * - dog`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`kawo\"nka`, true, false,
	})
}

func TestBadString(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`"cat *`)
	xt.Assert(len(r) == 1)

	xt.Assert(r[0] == letarette.Phrase{
		`cat *`, false, false,
	})
}

func TestDoubleDoubleQuotedString(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`""dog""`)
	xt.Assert(len(r) == 3)

	xt.Assert(r[0] == letarette.Phrase{
		``, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`dog`, false, false,
	})

	xt.Assert(r[2] == letarette.Phrase{
		``, false, false,
	})
}

func TestSingleQuoteExclusion(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`'WinkelWolt' "'Woff!"`)
	xt.Assert(len(r) == 2)

	xt.Assert(r[0] == letarette.Phrase{
		`WinkelWolt`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`'Woff!`, false, false,
	})
}

func TestParenthesisExclusion(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`(WinkelWolt) )))((( "()"`)
	xt.Assert(len(r) == 2)

	xt.Assert(r[0] == letarette.Phrase{
		`WinkelWolt`, false, false,
	})

	xt.Assert(r[1] == letarette.Phrase{
		`()`, false, false,
	})
}

func TestToString(t *testing.T) {
	xt := xt.X(t)

	r := letarette.ParseQuery(`"horse head" - nebula star * * -`)
	str := fmt.Sprintf("%s", r)
	xt.Assert(str == `["horse head" -nebula star*]`)
}

func TestReducePhraseList(t *testing.T) {
	xt := xt.X(t)

	phrases := letarette.ParseQuery(`rökare a a "b b" - angle "grinder u"*t`)
	xt.Assert(len(phrases) == 7)
	phrases = letarette.ReducePhraseList(phrases)
	xt.Assert(len(phrases) == 3)
	xt.Assert(phrases[0].Text == `rökare`)
	xt.Assert(phrases[1].Text == `angle`)
	xt.Assert(phrases[2].Text == `grinder`)
}

func TestCanonicalizePhraseList(t *testing.T) {
	xt := xt.X(t)

	listA := letarette.ParseQuery(`Yabba* -Dabba Doo Doo`)
	listB := letarette.ParseQuery(`-daBBa -dAbBa "DOO" "YABBA" *`)
	listA = letarette.CanonicalizePhraseList(listA)
	listB = letarette.CanonicalizePhraseList(listB)
	xt.DeepEqual(listA, listB)
}

func TestUnicodeCharacters(t *testing.T) {
	xt := xt.X(t)

	phrases := letarette.ParseQuery("rökare")
	xt.Assert(len(phrases) == 1)
	xt.Assert(phrases[0].Text == `rökare`)
}
