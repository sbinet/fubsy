package fubsy

import (
	"testing"
)

func TestScan_valid(t *testing.T) {
	lexer := NewLexer("nofile", []byte("]  [\"foo!bar\"\n ]"))
	lexer.scan()
	expect := []toktext{
		{"nofile", 1, ']', "]"},
		{"nofile", 1, '[', "["},
		{"nofile", 1, QSTRING, "\"foo!bar\""},
		{"nofile", 2, ']', "]"},
	}
	checkTokens(t, expect, lexer.tokens)
}

func TestScan_invalid(t *testing.T) {
	lexer := NewLexer("fwob", []byte("]]\n!-\"whee]\" x whizz\nbang"))
	lexer.scan()
	expect := []toktext{
		{"fwob", 1, ']', "]"},
		{"fwob", 1, ']', "]"},
		{"fwob", 2, BADTOKEN, "!-"},
		{"fwob", 2, QSTRING, "\"whee]\""},
		{"fwob", 2, BADTOKEN, "x"},
		{"fwob", 2, BADTOKEN, "whizz"},
		{"fwob", 3, BADTOKEN, "bang"},
		}
	checkTokens(t, expect, lexer.tokens)
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
