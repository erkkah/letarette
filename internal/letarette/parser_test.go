package letarette_test

import (
	"fmt"
	"testing"

	gta "gotest.tools/assert"

	"github.com/erkkah/letarette/internal/letarette"
)

func TestParsePlainPhrases(t *testing.T) {
	r := letarette.ParseQuery("cat dog banana")
	gta.Assert(t, len(r) == 3)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"dog"`, false, false,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`"banana"`, false, false,
	})
}

func TestIncludeExcludePhrases(t *testing.T) {
	r := letarette.ParseQuery("cat -dog banana - fishtank")
	gta.Assert(t, len(r) == 4)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"dog"`, false, true,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`"banana"`, false, false,
	})

	gta.Assert(t, r[3] == letarette.Phrase{
		`"fishtank"`, false, true,
	})
}

func TestWildcardPhrases(t *testing.T) {
	r := letarette.ParseQuery("cat* -dog * banana - fishtank")
	gta.Assert(t, len(r) == 4)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat"`, true, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"dog"`, true, true,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`"banana"`, false, false,
	})

	gta.Assert(t, r[3] == letarette.Phrase{
		`"fishtank"`, false, true,
	})
}

func TestEmbeddedAndFreeExcludes(t *testing.T) {
	r := letarette.ParseQuery("cat- cat-litter - -dog -")
	gta.Assert(t, len(r) == 3)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat-"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"cat-litter"`, false, false,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`"dog"`, false, true,
	})
}

func TestEmbeddedWildcard(t *testing.T) {
	r := letarette.ParseQuery(`cat* cat*litter "*dog*"`)
	gta.Assert(t, len(r) == 4)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat"`, true, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"cat"`, true, false,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`"litter"`, false, false,
	})

	gta.Assert(t, r[3] == letarette.Phrase{
		`"*dog*"`, false, false,
	})
}

func TestString(t *testing.T) {
	r := letarette.ParseQuery(`"cat - * - dog" "kawo\"nka"*`)
	gta.Assert(t, len(r) == 2)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat - * - dog"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"kawo\"nka"`, true, false,
	})
}

func TestBadString(t *testing.T) {
	r := letarette.ParseQuery(`"cat *`)
	gta.Assert(t, len(r) == 1)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"cat *`, false, false,
	})
}

func TestDoubleDoubleQuotedString(t *testing.T) {
	r := letarette.ParseQuery(`""dog""`)
	gta.Assert(t, len(r) == 3)

	gta.Assert(t, r[0] == letarette.Phrase{
		`""`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"dog"`, false, false,
	})

	gta.Assert(t, r[2] == letarette.Phrase{
		`""`, false, false,
	})
}

func TestSingleQuoteExclusion(t *testing.T) {
	r := letarette.ParseQuery(`'WinkelWolt' "'Woff!"`)
	gta.Assert(t, len(r) == 2)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"WinkelWolt"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"'Woff!"`, false, false,
	})
}

func TestParenthesisExclusion(t *testing.T) {
	r := letarette.ParseQuery(`(WinkelWolt) )))((( "()"`)
	gta.Assert(t, len(r) == 2)

	gta.Assert(t, r[0] == letarette.Phrase{
		`"WinkelWolt"`, false, false,
	})

	gta.Assert(t, r[1] == letarette.Phrase{
		`"()"`, false, false,
	})
}

func TestToString(t *testing.T) {
	r := letarette.ParseQuery(`"horse head" - nebula star * * -`)
	str := fmt.Sprintf("%s", r)
	gta.Assert(t, str == `["horse head" -"nebula" "star"*]`)
}
