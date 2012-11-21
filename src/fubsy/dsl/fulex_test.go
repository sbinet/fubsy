package dsl

import (
	"testing"
	"fmt"
	"strings"
	"github.com/stretchrcom/testify/assert"
)

func TestScan_valid_1(t *testing.T) {
	input := "xyz <foo*bar\n > # comment\n"
	expect := []minitok {
		{NAME, "xyz"},
		{'<', "<"},
		{FILEPATTERN, "foo*bar"},
		{'>', ">"},
		{EOL, "\n"},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 0,  3, 1,
		 4,  5, 1,
		 5, 12, 1,
		14, 15, 2,
		25, 26, 2)
}

func TestScan_eof(t *testing.T) {
	// input with no trailing newline generates a synthetic EOL
	// (text == "" so we can tell it was EOF)
	input := "main{  \n\"borf\""
	expect := []minitok {
		{NAME, "main"},
		{'{', "{"},
		{EOL, "\n"},
		{QSTRING, "\"borf\""},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 0,  4, 1,
		 4,  5, 1,
		 7,  8, 1,
		 8, 14, 2,
		14, 14, 2)
}

func TestScan_filelist(t *testing.T) {
	input := "bop { \n<**/*.[ch] [a-z]*.o\n>}"
	expect := []minitok {
		{NAME, "bop"},
		{'{', "{"},
		{EOL, "\n"},
		{'<', "<"},
		{FILEPATTERN, "**/*.[ch]"},
		{FILEPATTERN, "[a-z]*.o"},
		{'>', ">"},
		{'}', "}"},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 0,  3, 1,
		 4,  5, 1,
		 6,  7, 1,
		 7,  8, 2,
		 8, 17, 2,
		18, 26, 2,
		27, 28, 3,
		28, 29, 3,
		29, 29, 3)
}

func TestScan_valid_2(t *testing.T) {
	input := "main{\"foo\"<bar( )baz>} #ignore"
	expect := []minitok {
		{NAME, "main"},
		{'{', "{"},
		{QSTRING, "\"foo\""},
		{'<', "<"},
		{FILEPATTERN, "bar("},
		{FILEPATTERN, ")baz"},
		{'>', ">"},
		{'}', "}"},
		{EOL, ""},
	}
	assertScan(t, expect, scan(input))
}

func TestScan_keywords(t *testing.T) {
	input := "plugim import _import important .plugin\n"
	expect := []minitok {
		{NAME, "plugim"},
		{IMPORT, "import"},
		{NAME, "_import"},
		{NAME, "important"},
		{'.', "."},
		{PLUGIN, "plugin"},
		{EOL, "\n"},
	}
	assertScan(t, expect, scan(input))
}

func TestScan_inline_1(t *testing.T) {
	input := " plugin bob\n\n{{{yo\nhello\nthere\n}}}"
	expect := []minitok {
		{PLUGIN, "plugin"},
		{NAME, "bob"},
		{EOL, "\n"},
		{L3BRACE, "{{{"},
		{INLINE, "yo\nhello\nthere\n"},
		{R3BRACE, "}}}"},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 1,  7, 1,
		 8, 11, 1,
		11, 12, 1,
		13, 16, 3,
		16, 31, 3,
		31, 34, 6,
		34, 34, 6)

	sline, eline := tokens[4].location.linerange()
	assert.True(t, sline == 3 && eline == 5,
		fmt.Sprintf("expected sline == 3 (got %d) && eline == 5 (got %d)",
		sline, eline))
}

func TestScan_inline_2(t *testing.T) {
	// despite appearances, the original motivation for this test case
	// was newline (or indeed anything at all) after }}} -- I just
	// threw a bunch of punctuation into the inline text to be sure
	// that works too
	input := "\n\nplugin\ntim{{{ any!chars\"are\nallowed'here\n}}}\n"
	expect := []minitok {
		{PLUGIN, "plugin"},
		{EOL, "\n"},
		{NAME, "tim"},
		{L3BRACE, "{{{"},
		{INLINE, " any!chars\"are\nallowed'here\n"},
		{R3BRACE, "}}}"},
		{EOL, "\n"},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 2,  8, 3,
		 8,  9, 3,
		 9, 12, 4,
		12, 15, 4,
		15, 43, 4,
		43, 46, 6,
		46, 47, 6)
}

func TestScan_inline_unclosed_1(t *testing.T) {
	// bad input: make sure unclosed {{{ is not an infinite loop!
	input := " {{{bip\nbop!["
	expect := []minitok {
		{L3BRACE, "{{{"},
		{INLINE, "bip\nbop!["},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 1,  4, 1,
		 4, 13, 1,
		13, 13, 2)
}

func TestScan_inline_unclosed_2(t *testing.T) {
	// very similar to above, but with incomplete }}}
	input := " {{{bip\nbop![\n}}"
	expect := []minitok {
		{L3BRACE, "{{{"},
		{INLINE, "bip\nbop![\n}}"},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 1,  4, 1,
		 4, 16, 1,
		16, 16, 3)
}

func TestScan_inline_consecutive(t *testing.T) {
	input := "{{{\nbop\n}}}\n\n{{{meep\n}}}\n"
	expect := []minitok {
		{L3BRACE, "{{{"},
		{INLINE, "\nbop\n"},
		{R3BRACE, "}}}"},
		{EOL, "\n"},
		{L3BRACE, "{{{"},
		{INLINE, "meep\n"},
		{R3BRACE, "}}}"},
		{EOL, "\n"},
	}
	assertScan(t, expect, scan(input))
}

func TestScan_internal_newlines(t *testing.T) {
	input := "hello\n(\"beep\" +\n\"bop\"\n)\n foo"
	expect := []minitok {
		{NAME, "hello"},
		{EOL, "\n"},
		{'(', "("},
		{QSTRING, "\"beep\""},
		{'+', "+"},
		{QSTRING, "\"bop\""},
		{')', ")"},
		{EOL, "\n"},
		{NAME, "foo"},
		{EOL, ""},
	}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 0,  5, 1,
		 5,  6, 1,
		 6,  7, 2,
		 7, 13, 2,
		14, 15, 2,
		16, 21, 3,
		22, 23, 4,
		23, 24, 4,
		25, 28, 5,
		28, 28, 5)
}

func TestScan_invalid(t *testing.T) {
	input := "\n !-\"whee]\" whizz&^%\n?bang"
	input = "\n !-\"whee]\" whizz&^%\n?bang"
	expect := []minitok {
		{BADTOKEN, "!-"},
		{QSTRING, "\"whee]\""},
		{NAME, "whizz"},
		{BADTOKEN, "&^%"},
		{EOL, "\n"},
		{BADTOKEN, "?"},
		{NAME, "bang"},
		{EOL, ""},
		}
	tokens := scan(input)
	assertScan(t, expect, tokens)
	assertLocations(t, tokens,
		 2,  4, 2,
		 4, 11, 2,
		12, 17, 2,
		17, 20, 2,
		20, 21, 2,
		21, 22, 3,
		22, 26, 3,
		26, 26, 3)
}

func scan(input string) []token {
	scanner := NewScanner("", []byte(input))
	scanner.scan()
	return scanner.tokens
}

func assertScan(t *testing.T, expect []minitok, actual []token) {
	lasttok := actual[len(actual)-1]
	assert.Equal(t, EOF, lasttok.id,
		fmt.Sprintf("expected last token to be EOF, but got %d (%#v)",
		lasttok.id, lasttok.text))
	assertTokens(t, expect, actual[0:len(actual)-1])
}

func assertTokens(t *testing.T, expect []minitok, actual []token) {
	if len(expect) != len(actual) {
		tokens := make([]string, len(actual))
		for i, tok := range actual {
			tokens[i] = fmt.Sprintf("%#v", tok)
		}
		t.Fatalf("expected %d tokens, but got %d:\n%s",
			len(expect), len(actual), strings.Join(tokens, "\n"))
	}
	for i, etok := range expect {
		atok := minitok{id: actual[i].id, text: actual[i].text}
		assert.Equal(t, etok, atok,
			fmt.Sprintf("token %d: expected\n%#v\nbut got\n%#v", i, etok, atok))
	}
}

// locinfo is a sequence of start, end, startline triples
// (i.e. length must be N*3)
func assertLocations(t *testing.T, tokens []token, locinfo ...int) {
	tokens = tokens[:len(tokens) - 1] // ignore EOF token
	needlen := len(tokens) * 3
	if len(locinfo) != needlen {
		panic(fmt.Sprintf(
			"variable argument list must be of length %d " +
			"(3 per token: start, end, lineno), but got %d",
			needlen, len(locinfo)))
	}
	for i, tok := range tokens {
		prefix := fmt.Sprintf("token %d %#v: ", i, tok.text)

		start := locinfo[i*3 + 0]
		end := locinfo[i*3 + 1]
		startline := locinfo[i*3 + 2]
		assert.Equal(t, start, tok.location.start,
			fmt.Sprintf(prefix + "expected start == %d, but got %d",
			start, tok.location.start))
		assert.Equal(t, end, tok.location.end,
			fmt.Sprintf(prefix + "expected end == %d, but got %d",
			end, tok.location.end))

		sline, _ := tok.location.linerange()
		assert.Equal(t, startline, sline,
			fmt.Sprintf(prefix + "expected startline == %d, but got %d",
			startline, sline))
	}
}
