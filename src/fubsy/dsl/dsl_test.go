// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"bytes"
	"fubsy/testutils"
	"github.com/stretchrcom/testify/assert"
	"testing"
)

func Test_ParseString(t *testing.T) {
	script := "import foo.bar\nbleep {\na\n}"
	ast, err := ParseString("test", script)

	expect := &ASTRoot{children: []ASTNode{
		&ASTImport{plugin: []string{"foo", "bar"}},
		&ASTPhase{
			name: "bleep",
			children: []ASTNode{
				&ASTName{name: "a"}}}}}
	assert.Equal(t, 0, len(err))
	assertASTEqual(t, expect, ast)
}

// now with a syntax error
func Test_ParseString_invalid(t *testing.T) {
	script := "import foo,bar\nbleep {\n}\n"
	ast, err := ParseString("test", script)
	assert.Nil(t, ast)
	expect := "test:1: syntax error (near ',')"
	assertOneError(t, expect, err)
}

func TestParse_valid_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// dead simple: a single top-level element
	fn := testutils.Mkfile(tmpdir, "valid_1.fubsy", "main {\n<meep>\n\n}")

	expect := &ASTRoot{children: []ASTNode{
		&ASTPhase{name: "main", children: []ASTNode{
			&ASTFileFinder{patterns: []string{"meep"}}}}}}
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))
	assertASTEqual(t, expect, ast)
}

func TestParse_valid_sequence(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// sequence of top-level children
	fn := testutils.Mkfile(
		tmpdir,
		"valid_2.fubsy",
		"main {\n\"boo\"\n}\n"+
			"plugin foo {{{o'malley & friends\n}}}\n"+
			"blob {\n \"meep\"\n }")
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect := &ASTRoot{children: []ASTNode{
		&ASTPhase{
			name:     "main",
			children: []ASTNode{&ASTString{value: "boo"}}},
		&ASTInline{
			lang: "foo", content: "o'malley & friends\n"},
		&ASTPhase{
			name:     "blob",
			children: []ASTNode{&ASTString{value: "meep"}}},
	}}
	assertASTEqual(t, expect, ast)
}

func TestParse_internal_newlines(t *testing.T) {
	// newlines in a function call are invisible to the parser
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	fn := testutils.Mkfile(
		tmpdir,
		"newlines.fubsy",
		"main {\n"+
			//"  x(\n"+
			//"  a.b\n"+
			"  x("+
			"  a.b"+
			")\n"+
			"}")
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect := &ASTRoot{
		children: []ASTNode{
			&ASTPhase{
				name: "main",
				children: []ASTNode{
					&ASTFunctionCall{
						function: &ASTName{name: "x"},
						args: []ASTExpression{
							&ASTSelection{
								container: &ASTName{name: "a"},
								member:    "b",
							}}}},
			}}}
	assertASTEqual(t, expect, ast)
}

func TestParse_invalid_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// invalid: no closing rbracket
	fn := testutils.Mkfile(tmpdir, "invalid_1.fubsy", "main{  \n\"borf\"\n")
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near EOF)"
	assertOneError(t, expect, err)
}

func TestParse_invalid_2(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	// invalid: bad token
	fn := testutils.Mkfile(tmpdir, "invalid_2.fubsy", "main{\n *&! \"whizz\"\n}")
	_, err := Parse(fn)
	expect := fn + ":2: syntax error (near *&!)"
	assertOneError(t, expect, err)
	reset()
}

func assertOneError(t *testing.T, expect string, actual []error) {
	if len(actual) < 1 {
		t.Error("expected at least one error")
	} else if actual[0].Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual[0].Error())
	}
}

// this one tries to exercise many token types and grammar rules
func TestParse_omnibus_1(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	fn := testutils.Mkfile(tmpdir, "omnibus_1.fubsy",
		"# start with a comment\n"+
			"import foo\n"+
			"import foo.bar.baz\n"+
			"\n     "+
			"# blank lines are OK!\n"+
			"plugin funky {{{\n"+
			"any ol' crap! \"bring it on,\n"+
			"dude\" ...\n"+
			"}}}\n"+
			"main {\n"+
			"  a   =(\"foo\") + b\n"+
			"  c=(d.e)  ()\n"+
			"x.y.z\n"+
			"  <\n"+
			"    lib1/*.c\n"+
			"    lib2/**/*.c\n"+
			"  >\n"+
			"}\n")
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect :=
		"ASTRoot {\n" +
			"  ASTImport[foo]\n" +
			"  ASTImport[foo.bar.baz]\n" +
			"  ASTInline[funky] {{{\n" +
			"    any ol' crap! \"bring it on,\n" +
			"    dude\" ...\n" +
			"  }}}\n" +
			"  ASTPhase[main] {\n" +
			"    ASTAssignment[a]\n" +
			"      ASTAdd\n" +
			"      op1:\n" +
			"        ASTString[foo]\n" +
			"      op2:\n" +
			"        ASTName[b]\n" +
			"    ASTAssignment[c]\n" +
			"      ASTFunctionCall[d.e] (0 args)\n" +
			"    ASTSelection[x.y: z]\n" +
			"    ASTFileFinder[lib1/*.c lib2/**/*.c]\n" +
			"  }\n" +
			"}\n"
	var actual_ bytes.Buffer
	ast.Dump(&actual_, "")
	actual := actual_.String()
	if expect != actual {
		t.Errorf("expected AST:\n%s\nbut got:\n%s", expect, actual)
	}
}

func TestParse_omnibus_2(t *testing.T) {
	tmpdir, cleanup := testutils.Mktemp()
	defer cleanup()

	fn := testutils.Mkfile(tmpdir, "omnibus_2.fubsy",
		"\n"+
			"main {\n"+
			"  headers = <*.h>\n"+
			"\n"+
			"  a + b : <*.c> + headers {\n"+
			"    x = a\n"+
			"    \"cc $x\"\n"+
			"    f(a, b)\n"+
			"  }\n"+
			"}\n"+
			"")
	ast, err := Parse(fn)
	assert.Equal(t, 0, len(err))

	expect :=
		"ASTRoot {\n" +
			"  ASTPhase[main] {\n" +
			"    ASTAssignment[headers]\n" +
			"      ASTFileFinder[*.h]\n" +
			"    ASTBuildRule {\n" +
			"    targets:\n" +
			"      ASTAdd\n" +
			"      op1:\n" +
			"        ASTName[a]\n" +
			"      op2:\n" +
			"        ASTName[b]\n" +
			"    sources:\n" +
			"      ASTAdd\n" +
			"      op1:\n" +
			"        ASTFileFinder[*.c]\n" +
			"      op2:\n" +
			"        ASTName[headers]\n" +
			"    actions:\n" +
			"      ASTAssignment[x]\n" +
			"        ASTName[a]\n" +
			"      ASTString[cc $x]\n" +
			"      ASTFunctionCall[f] (2 args)\n" +
			"        ASTName[a]\n" +
			"        ASTName[b]\n" +
			"    }\n" +
			"  }\n" +
			"}\n"
	var actual_ bytes.Buffer
	ast.Dump(&actual_, "")
	actual := actual_.String()
	if expect != actual {
		t.Errorf("expected AST:\n%s\nbut got:\n%s", expect, actual)
	}
}
