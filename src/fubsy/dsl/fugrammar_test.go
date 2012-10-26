package dsl

import (
	"testing"
	"fmt"
	"bytes"
)

func Test_fuParse_valid_imports(t *testing.T) {
	tokens := []minitok{
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
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
			ImportNode{plugin: []string {"ding"}},
			ImportNode{plugin: []string {"dong", "ping", "whoo"}},
		}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_valid_phase(t *testing.T) {
	tokens := []minitok{
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
		{EOF, ""},
	}
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
	assertParses(t, &expect, tokens)
}

func Test_fuParse_empty_phase(t *testing.T) {
	tokens := []minitok{
		{NAME, "blah"},
		{'{', "{"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "blah",
				statements: []ASTNode {}},
	}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_expr_1(t *testing.T) {
	// parse "blorp { stuff = (foo); }"
	tokens := []minitok{
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
		{EOL, ""},
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "blorp",
				statements: []ASTNode {
					AssignmentNode{
						target: "stuff",
						expr: NameNode{"foo"},
	}}}}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_expr_2(t *testing.T) {
	// parse "floo { a + b + c() }
	tokens := []minitok{
		{NAME, "floo"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "a"},
		{'+', "+"},
		{NAME, "b"},
		{'+', "+"},
		{NAME, "c"},
		{'(', "("},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, ""},
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
		PhaseNode{
			name: "floo",
			statements: []ASTNode {
				AddNode{
					op1: AddNode{op1: NameNode{"a"}, op2: NameNode{"b"}},
					op2: FunctionCallNode{
						function: NameNode{"c"},
						args: []ExpressionNode {},
	}}}}}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_funccall_1(t *testing.T) {
	// parse "frob { foo(); }"
	tokens := []minitok{
		{NAME, "frob"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "foo"},
		{'(', "("},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
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
	assertParses(t, &expect, tokens)
}

// expected AST for the next couple of test cases
var _funccall_expect RootNode

func init() {
	_funccall_expect = RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "frob",
				statements: []ASTNode {
					FunctionCallNode{
						function: NameNode{"foo"},
						args: []ExpressionNode {
							StringNode{"bip"},
							NameNode{"x"},
	}}}}}}
}

func Test_fuParse_funccall_2(t *testing.T) {
	// parse:
	// frob {
	//   foo("bip", x)
    // }
	tokens := []minitok{
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
		{EOL, ""},
		{EOF, ""},
	}
	assertParses(t, &_funccall_expect, tokens)
}

func Test_fuParse_funccall_3(t *testing.T) {
	// trailing comma after last arg: same as above
	tokens := []minitok{
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
		{EOL, ""},
		{EOF, ""},
	}
	assertParses(t, &_funccall_expect, tokens)
}

func Test_fuParse_filelist(t *testing.T) {
	// parse "main { x = [**/*.c]; }"
	tokens := []minitok{
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
		{EOL, ""},
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "main",
				statements: []ASTNode {
					AssignmentNode{
						target: "x",
						expr: FileListNode{
							patterns: []string {
								"**/*.c",
	}}}}}}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_valid_inline(t *testing.T) {
	tokens := []minitok{
		{PLUGIN, "plugin"},
		{NAME, "whatever"},
		{L3BRACE, "{{{"},
		{INLINE, "beep!\"\nblam'" },
		{R3BRACE, "}}}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect := RootNode{
		elements: []ASTNode {
			InlineNode{lang: "whatever", content: "beep!\"\nblam'"}}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_invalid(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
		{QSTRING, "\"ding\"" },
		{'<', "<"},
		{EOF, ""},
	})
	tokens[0].lineno = 2		// ensure this makes it to the SyntaxError
	lexer := NewLexer(tokens)
	result := fuParse(lexer)
	assertParseFailure(t, result, lexer)
	assertSyntaxError(t, 2, "\"ding\"", lexer)
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
		{EOF, ""},
	})
	lexer := NewLexer(tokens)
	result := fuParse(lexer)
	assertParseFailure(t, result, lexer)
	assertSyntaxError(t, 0, "!#*$", lexer)
}

func reset() {
	// restore default parser state
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

func assertParses(t *testing.T, expect *RootNode, tokens []minitok) {
	reset()
	lexer := NewLexer(toklist(tokens))
	result := fuParse(lexer)
	//assertNoError(t, lexer.syntaxerror)
	if lexer.syntaxerror != nil {
		t.Fatal(fmt.Sprintf("unexpected syntax error: %s", lexer.syntaxerror))
	}
	assertTrue(t, result == 0, "fuParse() returned %d (expected 0)", result)
	assertTrue(t, lexer.ast != nil, "lexer.ast is nil (expected non-nil)")
	assertASTEquals(t, expect, lexer.ast)
}

func assertParseFailure(t *testing.T, result int, lexer *Lexer) {
	if result != 1 {
		t.Errorf("expected fuParse() to return 1, not %d", result)
	}
	if lexer.ast != nil {
		t.Errorf("expected nil lexer.ast, but it's: %v", lexer.ast)
	}
	assertNotNil(t, "lexer.syntaxerror", lexer.syntaxerror)
}

func assertSyntaxError(t *testing.T, lineno int, badtext string, lexer *Lexer) {
	actual := lexer.syntaxerror
	message := "syntax error"

	if !(actual.badtoken.lineno == lineno &&
		actual.badtoken.text == badtext &&
		actual.message == message) {

		expect := &SyntaxError{
			badtoken: &toktext{
				filename: lexer.syntaxerror.badtoken.filename,
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
