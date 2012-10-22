package fubsy

import (
	"testing"
	"fmt"
	"bytes"
)

func Test_fuParse_valid_imports(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{IMPORT, "import"},
		{NAME, "ding"},
		{IMPORT, "import"},
		{NAME, "dong"},
		{'.', "."},
		{NAME, "ping"},
		{'.', "."},
		{NAME, "whoo"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)

	expect := RootNode{
		elements: []ASTNode {
			ImportNode{plugin: []string {"ding"}},
			ImportNode{plugin: []string {"dong", "ping", "whoo"}},
		}}
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_valid_phase(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{NAME, "main"},
		{'{', "{"},
		{'[', "["},
		{QSTRING, "\"foo\""},
		{']', "]"},
		{'[', "["},
		{QSTRING, "\"bar\""},
		{']', "]"},
		{'}', "}"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)

	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
			name: "main",
			statements: []ASTNode {
					ListNode{values: []string {"foo"}},
					ListNode{values: []string {"bar"}},
	}}}}
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_empty_phase(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{NAME, "blah"},
		{'{', "{"},
		{'}', "}"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)

	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "blah",
				statements: []ASTNode {}},
		},
	}
	assertASTEquals(t, &expect, _ast)
}


func Test_fuParse_valid_inline(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{PLUGIN, "plugin"},
		{NAME, "whatever"},
		{L3BRACE, "{{{"},
		{INLINE, "beep!\"\nblam'" },
		{R3BRACE, "}}}"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)
	expect := RootNode{elements: []ASTNode {
			InlineNode{lang: "whatever", content: "beep!\"\nblam'"}}}
	assertASTEquals(t, &expect, _ast)
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
		{NAME, "blah"},
		{'{', "{"},
		{'[', "["},
		{QSTRING, "\"pop!\""},
		{BADTOKEN, "!#*$"},
		{']', "]"},
		{'}', "}"},
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

func assertParseSuccess(t *testing.T, result int) {
	assertNil(t, "_syntaxerror", _syntaxerror)
	//assertNoError(t, _syntaxerror)
	assertTrue(t, result == 0, "fuParse() returned %d (expected 0)", result)
	assertTrue(t, _ast != nil, "_ast is nil (expected non-nil)")
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

func assertASTEquals(t *testing.T, expect *RootNode, actual *RootNode) {
	if ! expect.Equal(*actual) {
		expectbuf := new(bytes.Buffer)
		actualbuf := new(bytes.Buffer)
		expect.Dump(expectbuf, "")
		actual.Dump(actualbuf, "")
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

func assertTrue(t *testing.T, p bool, fmt string, args ...interface{}) {
	if !p {
		t.Fatalf(fmt, args...)
	}
}

/*
// return a token in the form expected by parser.Parse()
func ptok(name string, value string) ParseTOKENTYPE {
	tok := ASTNode(ttok(name, value)) // ttok() is in fulex_test.go
	return ParseTOKENTYPE(&tok)
}
*/
