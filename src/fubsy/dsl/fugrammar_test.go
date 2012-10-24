package dsl

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
		{EOL, "\n"},
		{IMPORT, "import"},
		{NAME, "dong"},
		{'.', "."},
		{NAME, "ping"},
		{'.', "."},
		{NAME, "whoo"},
		{EOL, "\n"},
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
		{EOL, "\n"},
		{QSTRING, "\"foo\""},
		{EOL, "\n"},
		{NAME, "x"},
		{'=', "="},
		{QSTRING, "\"bar\""},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)

	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
			name: "main",
			statements: []ASTNode {
					StringNode{"foo"},
					AssignmentNode{
						target: "x",
						expr: StringNode{"bar"},
	}}}}}
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_empty_phase(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{NAME, "blah"},
		{'{', "{"},
		{'}', "}"},
		{EOL, "\n"},
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

func Test_fuParse_expr_1(t *testing.T) {
	reset()
	// parse "blorp { stuff = (foo); }"
	lexer := NewLexer(toklist([]minitok{
		{NAME, "blorp"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "stuff"},
		{'=', "="},
		{'(', "("},
		{NAME, "foo"},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "blorp",
				statements: []ASTNode {
					AssignmentNode{
						target: "stuff",
						expr: NameNode{"foo"},
				}},
	}}}
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_funccall_1(t *testing.T) {
	reset()
	// parse "frob { foo(); }"
	lexer := NewLexer(toklist([]minitok{
		{NAME, "frob"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "foo"},
		{'(', "("},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "frob",
				statements: []ASTNode {
					FunctionCallNode{
						function: NameNode{"foo"},
						args: []ExpressionNode {}},
				},
			},
	}}
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_funccall_2(t *testing.T) {
	reset()
	// parse:
	// frob {
	//   foo("bip", x)
    // }
	lexer := NewLexer(toklist([]minitok{
		{NAME, "frob"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "foo"},
		{'(', "("},
		{QSTRING, "\"bip\""},
		{',', ","},
		{NAME, "x"},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOF, ""},
	}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "frob",
				statements: []ASTNode {
					FunctionCallNode{
						function: NameNode{"foo"},
						args: []ExpressionNode {
							StringNode{"bip"},
							NameNode{"x"},
					}},
	}}}}
	assertASTEquals(t, &expect, _ast)

	// trailing comma after last arg: same result
	lexer = NewLexer(toklist([]minitok{
		{NAME, "frob"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "foo"},
		{'(', "("},
		{QSTRING, "\"bip\""},
		{',', ","},
		{NAME, "x"},
		{',', ","},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOF, ""},
	}))
	reset()
	result = fuParse(lexer)
	assertParseSuccess(t, result)
	assertASTEquals(t, &expect, _ast)
}

func Test_fuParse_filelist(t *testing.T) {
	reset()
	// parse "main { x = [**/*.c]; }"
	lexer := NewLexer(toklist([]minitok{
		{NAME, "main"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "x"},
		{'=', "="},
		{'<', "<"},
		{FILEPATTERN, "**/*.c"},
		{'>', ">"},
		{EOL, "\n"},
		{'}', "}"},
		{EOF, ""},
		}))

	result := fuParse(lexer)
	assertParseSuccess(t, result)
}

func Test_fuParse_valid_inline(t *testing.T) {
	reset()
	lexer := NewLexer(toklist([]minitok{
		{PLUGIN, "plugin"},
		{NAME, "whatever"},
		{L3BRACE, "{{{"},
		{INLINE, "beep!\"\nblam'" },
		{R3BRACE, "}}}"},
		{EOL, "\n"},
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
		{'<', "<"},
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
		{EOL, "\n"},
		{QSTRING, "\"pop!\""},
		{BADTOKEN, "!#*$"},
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
	//assertNoError(t, _syntaxerror)
	if _syntaxerror != nil {
		t.Fatal(fmt.Sprintf("unexpected syntax error: %s", _syntaxerror))
	}
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

func assertSyntaxError(t *testing.T, lineno int, badtext string) {
	actual := _syntaxerror
	message := "syntax error"

	if !(actual.badtoken.lineno == lineno &&
		actual.badtoken.text == badtext &&
		actual.message == message) {

		expect := &SyntaxError{
			badtoken: &toktext{
				filename: _syntaxerror.badtoken.filename,
				lineno: lineno,
				text: badtext},
			message: message}

		t.Errorf("expected syntax error:\n%s\nbut got:\n%s",
			expect, actual)
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
