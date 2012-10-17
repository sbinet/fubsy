package fubsy

import (
	"testing"
	"fmt"
	"bytes"
)

func Test_fuParse_valid(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{'[', "["},
		{QSTRING, "\"foo\"" },
		{']', "]"},
	}))

	result := fuParse(lexer)
	if result != 0 {
		t.Errorf("expected fuParse() to return 0, not %d", result)
	}
	assertNil(t, "_syntaxerror", _syntaxerror)
	if _ast == nil {
		t.Errorf("expected _ast to be set, but it's nil")
	}

	expect := RootNode{elements: []ASTNode {ListNode{values: []string {"foo"}}}}
	checkASTEquals(t, &expect, _ast)
}

func Test_fuParse_invalid(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
		{QSTRING, "\"ding\"" },
		{'[', "["},
	})
	tokens[0].lineno = 2		// ensure this makes it to the SyntaxError
	lexer := NewLexer(tokens)
	result := fuParse(lexer)
	assertParseFailure(t, result)
	assertSyntaxError(t, 2, "\"ding\"")
}

func Test_fuParse_badtoken(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
		{'[', "["},
		{QSTRING, "\"pop!\""},
		{BADTOKEN, "!#*$"},
		{']', "]"},
	})
	lexer := NewLexer(tokens)
	result := fuParse(lexer)
	assertParseFailure(t, result)
	assertSyntaxError(t, 0, "!#*$")
}

func reset() {
	// restore default parser state
	_lasttok = nil
	_ast = nil
	_syntaxerror = nil
}

// useful for constructing test data
type minitok struct {
	tok int
	text string
}

func toklist(tokens []minitok) []toktext {
	result := make([]toktext, len(tokens))
	for i, mtok := range tokens {
		result[i] = toktext{token: mtok.tok, text: mtok.text}
	}
	return result
}

func checkASTEquals(t *testing.T, expect *RootNode, actual *RootNode) {
	if ! expect.Equal(*actual) {
		expectbuf := new(bytes.Buffer)
		actualbuf := new(bytes.Buffer)
		expect.Dump(expectbuf, "")
		actual.Dump(actualbuf, "")
		fmt.Printf("expect.elements[0] = %#v\n", expect.elements[0])
		fmt.Printf("actual.elements[0] = %#v\n", actual.elements[0])
		t.Errorf("expected AST node:\n%sbut got:\n%s", expectbuf, actualbuf)
	}
}

func assertParseFailure(t *testing.T, result int) {
	if result != 1 {
		t.Errorf("expected fuParse() to return 1, not %d", result)
	}
	if _ast != nil {
		t.Errorf("expected nil _ast, but it's: %v", _ast)
	}
	assertNotNil(t, "_syntaxerror", _syntaxerror)
}

func assertSyntaxError(t *testing.T, lineno int, badtoken string) {
	expect := &SyntaxError{
		line: lineno,
		message: "syntax error",
		badtoken: badtoken,
	}
	if *expect != *_syntaxerror {
		t.Errorf("expected syntax error:\n%#v\nbut got:\n%#v", expect, _syntaxerror)
	}
}

func assertNil(t *testing.T, name string, p *SyntaxError) {
	if p != nil {
		t.Fatal(fmt.Sprintf("%s != nil (expected nil)", name))
	}
}

func assertNotNil(t *testing.T, name string, p *SyntaxError) {
	if p == nil {
		t.Fatal(fmt.Sprintf("%s == nil (expected non-nil)", name))
	}
}

/*
// return a token in the form expected by parser.Parse()
func ptok(name string, value string) ParseTOKENTYPE {
	tok := ASTNode(ttok(name, value)) // ttok() is in fulex_test.go
	return ParseTOKENTYPE(&tok)
}
*/
