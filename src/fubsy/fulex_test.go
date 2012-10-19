package fubsy

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

func TestScan_inline_1(t *testing.T) {
	input := " \n{{{yo\nhello\nthere\n}}}"
	expect := []toktext{
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
	input := "{{{ any!chars\"are\nallowed'here\n}}}\n"
	expect := []toktext{
		{"blop.txt", 1, L3BRACE, "{{{"},
		{"blop.txt", 1, INLINE, " any!chars\"are\nallowed'here\n"},
		{"blop.txt", 3, R3BRACE, "}}}"},
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
	input := "]]\n!-\"whee]\" x whizz\nbang"
	expect := []toktext{
		{"fwob", 1, ']', "]"},
		{"fwob", 1, ']', "]"},
		{"fwob", 2, BADTOKEN, "!-"},
		{"fwob", 2, QSTRING, "\"whee]\""},
		{"fwob", 2, BADTOKEN, "x"},
		{"fwob", 2, BADTOKEN, "whizz"},
		{"fwob", 3, BADTOKEN, "bang"},
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
