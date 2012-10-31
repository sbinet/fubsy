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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTImport{plugin: []string {"ding"}},
			ASTImport{plugin: []string {"dong", "ping", "whoo"}},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
			name: "main",
			statements: []ASTNode {
					ASTString{"foo"},
					ASTAssignment{
						target: "x",
						expr: ASTString{"bar"},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "blah",
				statements: []ASTNode {}},
	}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_globals(t *testing.T) {
	tokens := []minitok{
		{NAME, "v1"},
		{'=', "="},
		{QSTRING, "\"blobby!\""},
		{EOL, "\n"},
		{NAME, "v2"},
		{'=', "="},
		{'<', "<"},
		{FILEPATTERN, "*.h"},
		{FILEPATTERN, "*.hxx"},
		{'>', "<"},
		{'+', "+"},
		{'<', "<"},
		{FILEPATTERN, "*.c"},
		{'>', "<"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect := ASTRoot{
	elements: []ASTNode {
			ASTAssignment{
				target: "v1",
				expr: ASTString{"blobby!"},
			},
			ASTAssignment{
				target: "v2",
				expr: ASTAdd{
					op1: ASTFileList{patterns: []string {"*.h", "*.hxx"}},
					op2: ASTFileList{patterns: []string {"*.c"}},
			}},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "blorp",
				statements: []ASTNode {
					ASTAssignment{
						target: "stuff",
						expr: ASTName{"foo"},
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
	expect := ASTRoot{
		elements: []ASTNode {
		ASTPhase{
			name: "floo",
			statements: []ASTNode {
				ASTAdd{
					op1: ASTAdd{op1: ASTName{"a"}, op2: ASTName{"b"}},
					op2: ASTFunctionCall{
						function: ASTName{"c"},
						args: []ASTExpression {},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "frob",
				statements: []ASTNode {
					ASTFunctionCall{
						function: ASTName{"foo"},
						args: []ASTExpression {}},
				},
			},
	}}
	assertParses(t, &expect, tokens)
}

// expected AST for the next couple of test cases
var _funccall_expect ASTRoot

func init() {
	_funccall_expect = ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "frob",
				statements: []ASTNode {
					ASTFunctionCall{
						function: ASTName{"foo"},
						args: []ASTExpression {
							ASTString{"bip"},
							ASTName{"x"},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "main",
				statements: []ASTNode {
					ASTAssignment{
						target: "x",
						expr: ASTFileList{
							patterns: []string {
								"**/*.c",
	}}}}}}}
	assertParses(t, &expect, tokens)
}

func Test_fuParse_buildrule_1(t *testing.T) {
	// parse:
	// main {
	//   a: "x" {
	//     "foo bar"
	//     bip()
	//   }
	// }
	tokens := []minitok{
		{NAME, "main"},
		{'{', "{"},
		{EOL, "\n"},
		{NAME, "a"},
		{':', ":"},
		{QSTRING, "\"x\""},
		{'{', "{"},
		{EOL, "\n"},
		{QSTRING, "\"foo bar\""},
		{EOL, "\n"},
		{NAME, "bip"},
		{'(', "("},
		{')', ")"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, ""},
		{EOF, ""},
	}
	expect := ASTRoot{
		elements: []ASTNode {
			ASTPhase{
				name: "main",
				statements: []ASTNode {
					ASTBuildRule{
						targets: ASTName{"a"},
						sources: ASTString{"x"},
						actions: []ASTNode {
							ASTString{"foo bar"},
							ASTFunctionCall{
								function: ASTName{"bip"},
								args: []ASTExpression {},
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
	expect := ASTRoot{
		elements: []ASTNode {
			ASTInline{lang: "whatever", content: "beep!\"\nblam'"}}}
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
	parser := NewParser(tokens)
	result := fuParse(parser)
	assertParseFailure(t, result, parser)
	assertSyntaxError(t, 2, "\"ding\"", parser)
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
	parser := NewParser(tokens)
	result := fuParse(parser)
	assertParseFailure(t, result, parser)
	assertSyntaxError(t, 0, "!#*$", parser)
}

func Test_checkActions_ok(t *testing.T) {
	nodes := []ASTNode {
		ASTString{"foo"},
		ASTFunctionCall{function: ASTName{"foo"}},
		ASTAssignment{target: "x", expr: ASTFunctionCall{}},
		ASTFunctionCall{
			function: ASTName{"blah"},
			args: []ASTExpression {ASTString{"meep"}, ASTString{"beep"}}},
	}
	actual := checkActions(nodes)
	if len(actual) > 0 {
		t.Errorf("expected no errors, but got %v", actual)
	}
}

func Test_checkActions_bad(t *testing.T) {
	nodes := []ASTNode {
		ASTString{"foo bar"},			  // good
		ASTFileList{[]string {"*.java"}}, // bad
		ASTFunctionCall{},				  // good
		ASTBuildRule{					  // bad
			targets: ASTString{"t"},
			sources: ASTString{"s"},
			actions: []ASTNode {},
		},
		// hmmm: lots of expressions evaluate to a string -- why can't
		// I say cmd = "cc -o foo foo.c" outside a build rule, and then
		// reference cmd inside the build rule?
		ASTName{"blah"},		          // bad (currently)
	}
	expect := []SemanticError {
		SemanticError{node: nodes[1]},
		SemanticError{node: nodes[3]},
		SemanticError{node: nodes[4]},
	}
	actual := checkActions(nodes)
	assertTrue(t, len(actual) == len(expect),
		"expected %d errors, but got %d: %v",
		len(expect), len(actual), actual)
	for i, err := range expect {
		enode := err.node
		anode := actual[i].(SemanticError).node
		assertTrue(t, anode.Equal(enode),
			"bad node %d: expected %v, but got %v",
			i, enode, anode)
	}
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

func assertParses(t *testing.T, expect *ASTRoot, tokens []minitok) {
	reset()
	parser := NewParser(toklist(tokens))
	result := fuParse(parser)
	//assertNoError(t, parser.syntaxerror)
	if parser.syntaxerror != nil {
		t.Fatal(fmt.Sprintf("unexpected syntax error: %s", parser.syntaxerror))
	}
	assertTrue(t, result == 0, "fuParse() returned %d (expected 0)", result)
	assertTrue(t, parser.ast != nil, "parser.ast is nil (expected non-nil)")
	assertASTEquals(t, expect, parser.ast)
}

func assertParseFailure(t *testing.T, result int, parser *Parser) {
	if result != 1 {
		t.Errorf("expected fuParse() to return 1, not %d", result)
	}
	if parser.ast != nil {
		t.Errorf("expected nil parser.ast, but it's: %v", parser.ast)
	}
	assertNotNil(t, "parser.syntaxerror", parser.syntaxerror)
}

func assertSyntaxError(t *testing.T, lineno int, badtext string, parser *Parser) {
	actual := parser.syntaxerror
	message := "syntax error"

	if !(actual.badtoken.lineno == lineno &&
		actual.badtoken.text == badtext &&
		actual.message == message) {

		expect := &SyntaxError{
			badtoken: &toktext{
				filename: parser.syntaxerror.badtoken.filename,
				lineno: lineno,
				text: badtext},
			message: message}

		t.Errorf("expected syntax error:\n%s\nbut got:\n%s",
			expect, actual)
	}
}

func assertASTEquals(t *testing.T, expect *ASTRoot, actual *ASTRoot) {
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
