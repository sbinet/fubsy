package fubsy

import (
	"testing"
	"fmt"
	"strings"
)

func TestScan_empty(t *testing.T) {
	input := strings.NewReader("")
	tokens, err := Scan("test", input)
	checkError(t, nil, err)
	expect := make([]Token, 0)
	checkTokens(t, expect, tokens)
}

func TestScan_valid(t *testing.T) {
	input := strings.NewReader("\"foo! bar'baz\"][\"\"")
	tokens, err := Scan("test", input)
	checkError(t, nil, err)

	expect := []Token {
		ttok("qstring", "\"foo! bar'baz\""),
		ttok("rbracket", "]"),
		ttok("lbracket", "["),
		ttok("qstring", "\"\""),
	}
	checkTokens(t, expect, tokens)
}

func TestScan_invalid(t *testing.T) {
	input := strings.NewReader("]===\"yo\" pop !")
	expect := []Token {
		ttok("rbracket", "]"),
		ttok("qstring", "\"yo\""),
	}
	expecterr := ScanErrors{
		BadToken{"test", 1, []byte("===")},
		BadToken{"test", 1, []byte(" pop !")},
	}

	tokens, err := Scan("test", input)
	checkError(t, expecterr, err)
	checkTokens(t, expect, tokens)
}

// map token name ("qstring") to id
var tokmap map[string] int

func init() {
	tokmap = make(map[string] int)
	for i, tokdef := range tokenDefs {
		tokmap[tokdef.name] = i
	}
}

// return a minimal "testable" Token instance
func ttok(name string, value string) Token {
	id, ok := tokmap[name]
	if !ok {
		panic("bad token name: " + name)
	}
	return Token{id: id, value: value}
}

func checkError(t *testing.T, expect error, actual error) {
	if expect == nil && actual == nil {
		return
	} else if expect == nil && actual != nil {
		t.Error("unexpected error:", actual)
	} else if expect.Error() != actual.Error() {
		t.Error(fmt.Sprintf(
			"expected error message:\n%s\n" +
			"but got:\n%s",
			expect.Error(), actual.Error()))
	}
}

func checkTokens(t *testing.T, expect []Token, actual []Token) {
	if len(expect) != len(actual) {
		t.Error(fmt.Sprintf("expected %d tokens, but got %d: %v",
			len(expect), len(actual), actual))
	} else {
		for i, etok := range expect {
			atok := actual[i]
			if (etok.id != atok.id) || (etok.value != atok.value) {
				t.Error(fmt.Sprintf(
					"expected token %d (%v), but got %d (%v)",
					etok.id, etok.value, atok.id, atok.value))
			}
		}
	}
}
