package fubsy

import (
	"testing"
	"fmt"
	"bytes"
)

func TestParse_valid(t *testing.T) {
	lexer := &DummyLexer{tokens: toklist([]minitok{
		{'[', "["},
		{QSTRING, "\"foo\"" },
		{']', "]"},
	})}

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

func TestParse_invalid(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
			{QSTRING, "\"ding\"" },
			{'[', "["},
	})
	tokens[0].lineno = 2		// ensure this makes it to the SyntaxError
	lexer := &DummyLexer{tokens: tokens}
	result := fuParse(lexer)
	if result != 1 {
		t.Errorf("expected fuParse() to return 1, not %d", result)
	}
	if _ast != nil {
		t.Errorf("expected nil _ast, but it's: %v", _ast)
	}
	assertNotNil(t, "_syntaxerror", _syntaxerror)
	expect := &SyntaxError{
		line: 2,
		message: "syntax error",
		badtoken: "\"ding\"",
	}
	if *expect != *_syntaxerror {
		t.Errorf("expected syntax error:\n%#v\nbut got:\n%#v", expect, _syntaxerror)
	}
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
