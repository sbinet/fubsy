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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTImport{plugin: []string {"ding"}},
			&ASTImport{plugin: []string {"dong", "ping", "whoo"}},
		}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
			name: "main",
			children: []ASTNode {
					&ASTString{value: "foo"},
					&ASTAssignment{
						target: "x",
						expr: &ASTString{value: "bar"},
	}}}}}
	assertParses(t, expect, tokens)
}

func Test_fuParse_empty_phase(t *testing.T) {
	tokens := []minitok{
		{NAME, "blah"},
		{'{', "{"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "blah",
				children: []ASTNode {}},
	}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
	children: []ASTNode {
			&ASTAssignment{
				target: "v1",
				expr: &ASTString{value: "blobby!"},
			},
			&ASTAssignment{
				target: "v2",
				expr: &ASTAdd{
					op1: &ASTFileList{patterns: []string {"*.h", "*.hxx"}},
					op2: &ASTFileList{patterns: []string {"*.c"}},
			}},
	}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "blorp",
				children: []ASTNode {
					&ASTAssignment{
						target: "stuff",
						expr: &ASTName{name: "foo"},
	}}}}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
		&ASTPhase{
			name: "floo",
			children: []ASTNode {
				&ASTAdd{
					op1: &ASTAdd{
						op1: &ASTName{name: "a"},
						op2: &ASTName{name: "b"}},
					op2: &ASTFunctionCall{
						function: &ASTName{name: "c"},
						args: []ASTExpression {},
	}}}}}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "frob",
				children: []ASTNode {
					&ASTFunctionCall{
						function: &ASTName{name: "foo"},
						args: []ASTExpression {}},
				},
			},
	}}
	assertParses(t, expect, tokens)
}

// expected AST for the next couple of test cases
var _funccall_expect *ASTRoot

func init() {
	_funccall_expect = &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "frob",
				children: []ASTNode {
					&ASTFunctionCall{
						function: &ASTName{name: "foo"},
						args: []ASTExpression {
							&ASTString{value: "bip"},
							&ASTName{name: "x"},
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
	assertParses(t, _funccall_expect, tokens)
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
	assertParses(t, _funccall_expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "main",
				children: []ASTNode {
					&ASTAssignment{
						target: "x",
						expr: &ASTFileList{
							patterns: []string {
								"**/*.c",
	}}}}}}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{
				name: "main",
				children: []ASTNode {
					&ASTBuildRule{
						targets: &ASTName{name: "a"},
						sources: &ASTString{value: "x"},
						children: []ASTNode {
							&ASTString{value: "foo bar"},
							&ASTFunctionCall{
								function: &ASTName{name: "bip"},
								args: []ASTExpression {},
	}}}}}}}
	assertParses(t, expect, tokens)
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
	expect := &ASTRoot{
		children: []ASTNode {
			&ASTInline{lang: "whatever", content: "beep!\"\nblam'"}}}
	assertParses(t, expect, tokens)
}

func Test_fuParse_invalid(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
		{QSTRING, "\"ding\"" },
		{'<', "<"},
		{EOF, ""},
	})
	//tokens[0].lineno = 2		// ensure this makes it to the SyntaxError
	parser := NewParser(tokens)
	result := fuParse(parser)
	assertParseFailure(t, result, parser)
	assertSyntaxError(t, "\"ding\"", parser)
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
	assertSyntaxError(t, "!#*$", parser)
}

// Ensure that token locations propagate to AST node locations.
func Test_fuParse_ast_locations(t *testing.T) {
	// notional input:
	// "main{\nfoo = bar(\n  )\n}\n"

	fi := &fileinfo{"foo.txt", []int {0, 6, 17, 21, 23, 24}}
	tokens := []token {
		{location{fi, 0, 4}, NAME, "main"},
		{location{fi, 4, 5}, '{', "{"},
		{location{fi, 5, 6}, EOL, "\n"},
		{location{fi, 6, 9}, NAME, "foo"},
		{location{fi, 10, 11}, '=', "="},
		{location{fi, 12, 15}, NAME, "bar"},
		{location{fi, 15, 16}, '(', "("},
		{location{fi, 19, 20}, ')', ")"},
		{location{fi, 20, 21}, EOL, "\n"},
		{location{fi, 21, 22}, '}', "}"},
		{location{fi, 22, 23}, EOL, "\n"},
		{location{fi, 23, 23}, EOF, ""},
	}

	parser := NewParser(tokens)
	result := fuParse(parser)
	assertTrue(t, result == 0 && parser.ast != nil,
		"parse failed: result == %d, parser.ast = %v", result, parser.ast)

	assertLocation := func(node ASTNode, estart int, eend int) {
		location := node.Location()
		assertTrue(t, location.start == estart && location.end == eend,
			"%T: expected location %d:%d, but got %d:%d",
			node, estart, eend, location.start, location.end)
	}

	root := parser.ast
	phase := root.children[0].(*ASTPhase)
	assign := phase.children[0].(*ASTAssignment)
	rhs := assign.expr.(*ASTFunctionCall)
	name := rhs.function.(*ASTName)
	assertLocation(name, 12, 15)
	assertLocation(rhs, 12, 20)
	assertLocation(assign, 6, 20)
	assertLocation(phase, 0, 22)
	assertLocation(root, 0, 22)
}

func reset() {
	// restore default parser state
}

// useful for constructing test data
type minitok struct {
	id int
	text string
}

func toklist(tokens []minitok) []token {
	result := make([]token, len(tokens))
	for i, mtok := range tokens {
		result[i] = token{id: mtok.id, text: mtok.text}
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

func assertSyntaxError(t *testing.T, badtext string, parser *Parser) {
	actual := parser.syntaxerror
	message := "syntax error"

	if !(actual.badtoken.text == badtext && actual.message == message) {
		expect := &SyntaxError{
			badtoken: &token{text: badtext},
			message: message}

		t.Errorf("expected syntax error:\n%s\nbut got:\n%s",
			expect, actual)
	}
}

func assertASTEquals(t *testing.T, expect *ASTRoot, actual *ASTRoot) {
	if ! expect.Equal(actual) {
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
