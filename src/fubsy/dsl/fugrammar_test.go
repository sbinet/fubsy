// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"bytes"

	"github.com/stretchrcom/testify/assert"

	"testing"
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
		children: []ASTNode{
			&ASTImport{plugin: []string{"ding"}},
			&ASTImport{plugin: []string{"dong", "ping", "whoo"}},
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
		children: []ASTNode{
			&ASTPhase{
				name: "main",
				children: []ASTNode{
					&ASTString{value: "foo"},
					&ASTAssignment{
						target: "x",
						expr:   &ASTString{value: "bar"},
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
		children: []ASTNode{
			&ASTPhase{
				name:     "blah",
				children: []ASTNode{}},
		}}
	assertParses(t, expect, tokens)

	// same thing, with one or more empty lines in the body
	// (the lexer collapses multiple newlines into a single EOL token)
	tokens = []minitok{
		{NAME, "blah"},
		{'{', "{"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
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
		children: []ASTNode{
			&ASTPhase{
				name: "blorp",
				children: []ASTNode{
					&ASTAssignment{
						target: "stuff",
						expr:   &ASTName{name: "foo"},
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
		children: []ASTNode{
			&ASTPhase{
				name: "floo",
				children: []ASTNode{
					&ASTAdd{
						op1: &ASTAdd{
							op1: &ASTName{name: "a"},
							op2: &ASTName{name: "b"}},
						op2: &ASTFunctionCall{
							function: &ASTName{name: "c"},
							args:     []ASTExpression{},
						}}}}}}
	assertParses(t, expect, tokens)
}

func Test_FuParse_list(t *testing.T) {
	var tokens []minitok
	var expect *ASTRoot

	// empty list: "[]"
	tokens = []minitok{
		{NAME, "murp"},
		{'{', "{"},
		{EOL, "\n"},
		{'[', "["},
		{']', "]"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect = &ASTRoot{
		children: []ASTNode{
			&ASTPhase{
				name: "murp",
				children: []ASTNode{
					&ASTList{elements: []ASTExpression{}},
				}}}}
	assertParses(t, expect, tokens)

	// parse "[a, b(), foo + bar,]"
	tokens = []minitok{
		{NAME, "murp"},
		{'{', "{"},
		{EOL, "\n"},
		{'[', "["},
		{NAME, "a"},
		{',', ","},
		{NAME, "b"},
		{'(', "("},
		{')', ")"},
		{',', ","},
		{NAME, "foo"},
		{'+', "+"},
		{NAME, "bar"},
		{',', ","},
		{']', "]"},
		{EOL, "\n"},
		{'}', "}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect = &ASTRoot{
		children: []ASTNode{
			&ASTPhase{
				name: "murp",
				children: []ASTNode{
					&ASTList{
						elements: []ASTExpression{
							&ASTName{name: "a"},
							&ASTFunctionCall{
								function: &ASTName{name: "b"},
								args:     []ASTExpression{},
							},
							&ASTAdd{
								op1: &ASTName{name: "foo"},
								op2: &ASTName{name: "bar"},
							},
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
		children: []ASTNode{
			&ASTPhase{
				name: "frob",
				children: []ASTNode{
					&ASTFunctionCall{
						function: &ASTName{name: "foo"},
						args:     []ASTExpression{}},
				}}}}
	assertParses(t, expect, tokens)
}

// expected AST for the next couple of test cases
var _funccall_expect *ASTRoot

func init() {
	_funccall_expect = &ASTRoot{
		children: []ASTNode{
			&ASTPhase{
				name: "frob",
				children: []ASTNode{
					&ASTFunctionCall{
						function: &ASTName{name: "foo"},
						args: []ASTExpression{
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

func Test_fuParse_filefinder(t *testing.T) {
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
		children: []ASTNode{
			&ASTPhase{
				name: "main",
				children: []ASTNode{
					&ASTAssignment{
						target: "x",
						expr: &ASTFileFinder{
							patterns: []string{
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
		children: []ASTNode{
			&ASTPhase{
				name: "main",
				children: []ASTNode{
					&ASTBuildRule{
						targets: &ASTName{name: "a"},
						sources: &ASTString{value: "x"},
						children: []ASTNode{
							&ASTString{value: "foo bar"},
							&ASTFunctionCall{
								function: &ASTName{name: "bip"},
								args:     []ASTExpression{},
							}}}}}}}
	assertParses(t, expect, tokens)
}

func Test_fuParse_valid_inline(t *testing.T) {
	tokens := []minitok{
		{PLUGIN, "plugin"},
		{NAME, "whatever"},
		{L3BRACE, "{{{"},
		{INLINE, "beep!\"\nblam'"},
		{R3BRACE, "}}}"},
		{EOL, "\n"},
		{EOF, ""},
	}
	expect := &ASTRoot{
		children: []ASTNode{
			&ASTInline{lang: "whatever", content: "beep!\"\nblam'"}}}
	assertParses(t, expect, tokens)
}

func Test_fuParse_invalid(t *testing.T) {
	reset()
	tokens := toklist([]minitok{
		{QSTRING, "\"ding\""},
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

	fi := &fileinfo{"foo.txt", []int{0, 6, 17, 21, 23, 24}}
	tokens := []token{
		{FileLocation{fi, 0, 4}, NAME, "main"},
		{FileLocation{fi, 4, 5}, '{', "{"},
		{FileLocation{fi, 5, 6}, EOL, "\n"},
		{FileLocation{fi, 6, 9}, NAME, "foo"},
		{FileLocation{fi, 10, 11}, '=', "="},
		{FileLocation{fi, 12, 15}, NAME, "bar"},
		{FileLocation{fi, 15, 16}, '(', "("},
		{FileLocation{fi, 19, 20}, ')', ")"},
		{FileLocation{fi, 20, 21}, EOL, "\n"},
		{FileLocation{fi, 21, 22}, '}', "}"},
		{FileLocation{fi, 22, 23}, EOL, "\n"},
		{FileLocation{fi, 23, 23}, EOF, ""},
	}

	parser := NewParser(tokens)
	result := fuParse(parser)
	assert.True(t, result == 0 && parser.ast != nil,
		"parse failed: result == %d, parser.ast = %v",
		result, parser.ast)

	assertLocation := func(node ASTNode, estart int, eend int) {
		location := node.Location()
		assert.True(t, location.(FileLocation).start == estart,
			"%T: expected location %d:%d, but got %v",
			node, estart, eend, location)
		assert.True(t, location.(FileLocation).end == eend,
			"%T: expected location %d:%d, but got %v",
			node, estart, eend, location)
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
	id   int
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

	assert.Nil(t, parser.syntaxerror)
	assert.Equal(t, 0, result)
	assert.NotNil(t, parser.ast)
	assertASTEqual(t, expect, parser.ast)
}

func assertParseFailure(t *testing.T, result int, parser *Parser) {
	assert.Equal(t, 1, result)
	assert.Nil(t, parser.ast)
	assert.NotNil(t, parser.syntaxerror)
}

func assertSyntaxError(t *testing.T, badtext string, parser *Parser) {
	actual := parser.syntaxerror
	message := "syntax error"

	assert.Equal(t, message, actual.message)
	assert.Equal(t, badtext, actual.badtoken.text,
		"bad token text: expected %#v, but got %#v",
		badtext, actual.badtoken.text)
}

func assertASTEqual(t *testing.T, expect *ASTRoot, actual *ASTRoot) {
	if !expect.Equal(actual) {
		expectbuf := new(bytes.Buffer)
		actualbuf := new(bytes.Buffer)
		expect.Dump(expectbuf, "")
		actual.Dump(actualbuf, "")
		t.Errorf("expected AST node:\n%sbut got:\n%s", expectbuf, actualbuf)
	}
}
