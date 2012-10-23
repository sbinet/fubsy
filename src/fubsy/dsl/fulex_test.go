package dsl

import (
	"testing"
)

func TestScan_valid_1(t *testing.T) {
	input := "]  [\"foo!bar\"\n ]"
	expect := []toktext{
		{"nofile", 1, ']', "]"},
		{"nofile", 1, '[', "["},
		{"nofile", 1, QSTRING, "\"foo!bar\""},
		{"nofile", 2, ']', "]"},
	}
	assertScan(t, expect, "nofile", input)
}

func TestScan_valid_2(t *testing.T) {
	input := "main{[\"foo\"][\"bar\"]}"
	expect := []toktext{
		{"a.txt", 1, NAME, "main"},
		{"a.txt", 1, '{', "{"},
		{"a.txt", 1, '[', "["},
		{"a.txt", 1, QSTRING, "\"foo\""},
		{"a.txt", 1, ']', "]"},
		{"a.txt", 1, '[', "["},
		{"a.txt", 1, QSTRING, "\"bar\""},
		{"a.txt", 1, ']', "]"},
		{"a.txt", 1, '}', "}"},
	}
	assertScan(t, expect, "a.txt", input)
}

func TestScan_keywords(t *testing.T) {
	input := "plugim\nimport\n_import\nimportant\n.plugin\n"
	expect := []toktext{
		{"b.txt", 1, NAME, "plugim"},
		{"b.txt", 2, IMPORT, "import"},
		{"b.txt", 3, NAME, "_import"},
		{"b.txt", 4, NAME, "important"},
		{"b.txt", 5, '.', "."},
		{"b.txt", 5, PLUGIN, "plugin"},
	}
	assertScan(t, expect, "b.txt", input)
}

func TestScan_inline_1(t *testing.T) {
	input := " plugin bob\n{{{yo\nhello\nthere\n}}}"
	expect := []toktext{
		{"blop.txt", 1, PLUGIN, "plugin"},
		{"blop.txt", 1, NAME, "bob"},
		{"blop.txt", 2, L3BRACE, "{{{"},
		{"blop.txt", 2, INLINE, "yo\nhello\nthere\n"},
		{"blop.txt", 5, R3BRACE, "}}}"},
	}
	assertScan(t, expect, "blop.txt", input)
}

func TestScan_inline_2(t *testing.T) {
	// despite appearances, the original motivation for this test case
	// was newline (or indeed anything at all) after }}} -- I just
	// threw a bunch of punctuation into the inline text to be sure
	// that works too
	input := "plugin\ntim{{{ any!chars\"are\nallowed'here\n}}}\n"
	expect := []toktext{
		{"blop.txt", 1, PLUGIN, "plugin"},
		{"blop.txt", 2, NAME, "tim"},
		{"blop.txt", 2, L3BRACE, "{{{"},
		{"blop.txt", 2, INLINE, " any!chars\"are\nallowed'here\n"},
		{"blop.txt", 4, R3BRACE, "}}}"},
	}
	assertScan(t, expect, "blop.txt", input)
}

func TestScan_inline_open(t *testing.T) {
	// bad input: unclosed {{{ (should be a syntax error, not an
	// infinite loop!) (hmmm: would be nice to report the trailing
	// inline contents as a BADTOKEN; might give a better syntax error)
	input := "] {{{bip\nbop!["
	expect := []toktext{
		{"foo", 1, ']', "]"},
		{"foo", 1, L3BRACE, "{{{"},
	}
	assertScan(t, expect, "foo", input)

	// same result on incomplete }}}
	input += "\n}}"
	assertScan(t, expect, "foo", input)
}

func TestScan_invalid(t *testing.T) {
	input := "]]\n!-\"whee]\" whizz&^%\n?bang"
	expect := []toktext{
		{"fwob", 1, ']', "]"},
		{"fwob", 1, ']', "]"},
		{"fwob", 2, BADTOKEN, "!-"},
		{"fwob", 2, QSTRING, "\"whee]\""},
		{"fwob", 2, NAME, "whizz"},
		{"fwob", 2, BADTOKEN, "&^%"},
		{"fwob", 3, BADTOKEN, "?"},
		{"fwob", 3, NAME, "bang"},
		}
	assertScan(t, expect, "fwob", input)
}

func assertScan(t *testing.T, expect []toktext, filename string, input string) {
	scanner := NewScanner(filename, []byte(input))
	scanner.scan()
	assertTokens(t, expect, scanner.tokens)
}

func assertTokens(t *testing.T, expect []toktext, actual []toktext) {
	if len(expect) != len(actual) {
		t.Fatalf("expected %d tokens, but got %d",
			len(expect), len(actual))
	}
	for i, etok := range expect {
		atok := actual[i]
		if etok != atok {
			t.Errorf("token %d: expected\n%#v\nbut got\n%#v", i, etok, atok)
		}
	}

}
