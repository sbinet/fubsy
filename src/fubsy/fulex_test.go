package fubsy

import (
	"testing"
)

func TestScan_valid(t *testing.T) {
	scanner := NewScanner("nofile", []byte("]  [\"foo!bar\"\n ]"))
	scanner.scan()
	expect := []toktext{
		{"nofile", 1, ']', "]"},
		{"nofile", 1, '[', "["},
		{"nofile", 1, QSTRING, "\"foo!bar\""},
		{"nofile", 2, ']', "]"},
	}
	checkTokens(t, expect, scanner.tokens)
}

func TestScan_inline_1(t *testing.T) {
	scanner := NewScanner(
		"blop.txt", []byte(" \n{{{yo\nhello\nthere\n}}}"))
	scanner.scan()
	expect := []toktext{
		{"blop.txt", 2, L3BRACE, "{{{"},
		{"blop.txt", 2, INLINE, "yo\nhello\nthere\n"},
		{"blop.txt", 5, R3BRACE, "}}}"},
	}
	checkTokens(t, expect, scanner.tokens)
}

func TestScan_inline_2(t *testing.T) {
	// despite appearances, the original motivation for this test case
	// was newline (or indeed anything at all) after }}} -- I just
	// threw a bunch of punctuation into the inline text to be sure
	// that works too
	scanner := NewScanner(
		"blop.txt", []byte("{{{ any!chars\"are\nallowed'here\n}}}\n"))
	scanner.scan()
	expect := []toktext{
		{"blop.txt", 1, L3BRACE, "{{{"},
		{"blop.txt", 1, INLINE, " any!chars\"are\nallowed'here\n"},
		{"blop.txt", 3, R3BRACE, "}}}"},
	}
	checkTokens(t, expect, scanner.tokens)
}

func TestScan_inline_open(t *testing.T) {
	// bad input: unclosed {{{ (should be a syntax error, not an
	// infinite loop!) (hmmm: would be nice to report the trailing
	// inline contents as a BADTOKEN; might give a better syntax error)
	scanner := NewScanner("foo", []byte("] {{{bip\nbop!["))
	scanner.scan()
	expect := []toktext{
		{"foo", 1, ']', "]"},
		{"foo", 1, L3BRACE, "{{{"},
	}
	checkTokens(t, expect, scanner.tokens)

	// same result on incomplete }}}
	scanner = NewScanner("foo", append(scanner.input, "\n}}"...))
	scanner.scan()
	checkTokens(t, expect, scanner.tokens)
}

func TestScan_invalid(t *testing.T) {
	scanner := NewScanner("fwob", []byte("]]\n!-\"whee]\" x whizz\nbang"))
	scanner.scan()
	expect := []toktext{
		{"fwob", 1, ']', "]"},
		{"fwob", 1, ']', "]"},
		{"fwob", 2, BADTOKEN, "!-"},
		{"fwob", 2, QSTRING, "\"whee]\""},
		{"fwob", 2, BADTOKEN, "x"},
		{"fwob", 2, BADTOKEN, "whizz"},
		{"fwob", 3, BADTOKEN, "bang"},
		}
	checkTokens(t, expect, scanner.tokens)
}

func checkTokens(t *testing.T, expect []toktext, actual []toktext) {
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
