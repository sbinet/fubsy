package dsl

import (
	"testing"
	"os"
	"io/ioutil"
	"bytes"
	"path"

	"fubsy/testutils"
)

func TestParse_valid_1(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// dead simple: a single top-level element
	fn := mkfile(tmpdir, "valid_1.fubsy", "main {\n<meep>\n\n}")

	expect := RootNode{elements: []ASTNode {
			PhaseNode{name: "main", statements: []ASTNode {
					FileListNode{patterns: []string {"meep"}}}}}}
	ast, err := Parse(fn)
	testutils.AssertNoError(t, err)
	assertASTEquals(t, &expect, ast)
}

func TestParse_valid_sequence(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// sequence of top-level elements
	fn := mkfile(
		tmpdir,
		"valid_2.fubsy",
		"main {\n\"boo\"\n}\n" +
		"plugin foo {{{o'malley & friends\n}}}\n" +
		"blob {\n \"meep\"\n }")
	ast, err := Parse(fn)
	testutils.AssertNoError(t, err)

	expect := RootNode{elements: []ASTNode {
			PhaseNode{
				name: "main",
				statements: []ASTNode {StringNode{"boo"}}},
			InlineNode{
				lang: "foo", content: "o'malley & friends\n"},
			PhaseNode{
				name: "blob",
				statements: []ASTNode {StringNode{"meep"}}},
	}}
	assertASTEquals(t, &expect, ast)
}

func TestParse_internal_newlines(t *testing.T) {
	// newlines in a function call are invisible to the parser
	tmpdir, cleanup := mktemp()
	defer cleanup()

	fn := mkfile(
		tmpdir,
		"newlines.fubsy",
		"main {\n"+
			"  x(\n"+
			"  a.b\n"+
			")\n"+
			"}")
	ast, err := Parse(fn)
	testutils.AssertNoError(t, err)

	expect := RootNode{
		elements: []ASTNode {
			PhaseNode{
				name: "main",
				statements: []ASTNode {
					FunctionCallNode{
						function: NameNode{"x"},
						args: []ExpressionNode {
							SelectionNode{
								container: NameNode{"a"},
								member: "b",
				}}}},
	}}}
	assertASTEquals(t, &expect, ast)
}

func TestParse_invalid_1(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: no closing rbracket
	fn := mkfile(tmpdir, "invalid_1.fubsy", "main{  \n\"borf\"\n")
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near EOF)"
	testutils.AssertError(t, expect, err)
}

func TestParse_invalid_2(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: bad token
	fn := mkfile(tmpdir, "invalid_2.fubsy", "main{\n *&! \"whizz\"\n}")
	_, err := Parse(fn)
	expect := fn + ":2: syntax error (near *&!)"
	testutils.AssertError(t, expect, err)
	reset()
}

// this one tries to exercise every token type and grammar rule
func TestParse_everything(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	fn := mkfile(tmpdir, "everything.fubsy",
		"# start with a comment\n" +
		"import foo\n" +
		"import foo.bar.baz\n" +
		"\n     " +
		"# blank lines are OK!\n" +
		"plugin funky {{{\n" +
		"any ol' crap! \"bring it on,\n" +
		"dude\" ...\n" +
		"}}}\n" +
		"SRC = <lib/*.c>\n" +
		"main {\n" +
		"  a   =(\"foo\") + b\n" +
		"  c=(d.e)  ()\n" +
		"x.y.z\n" +
		"  <\n" +
		"    lib1/*.c\n" +
		"    lib2/**/*.c\n" +
		"  >\n" +
		"}\n",
	)
	ast, err := Parse(fn)
	testutils.AssertNoError(t, err)

	expect :=
		"RootNode {\n" +
		"  ImportNode[foo]\n" +
		"  ImportNode[foo.bar.baz]\n" +
		"  InlineNode[funky] {{{\n" +
		"    any ol' crap! \"bring it on,\n" +
		"    dude\" ...\n" +
		"  }}}\n" +
		"  AssignmentNode[SRC]\n" +
		"    FileListNode[lib/*.c]\n" +
		"  PhaseNode[main] {\n" +
		"    AssignmentNode[a]\n" +
		"      AddNode\n" +
		"      op1:\n" +
		"        StringNode[foo]\n" +
		"      op2:\n" +
		"        NameNode[b]\n" +
		"    AssignmentNode[c]\n" +
		"      FunctionCallNode[d.e] (0 args)\n" +
		"    SelectionNode[x.y: z]\n" +
		"    FileListNode[lib1/*.c lib2/**/*.c]\n" +
		"  }\n" +
		"}\n"
	var actual_ bytes.Buffer
	ast.Dump(&actual_, "")
	actual := actual_.String()
	if expect != actual {
		t.Errorf("expected AST:\n%s\nbut got:\n%s", expect, actual)
	}
}

func mktemp() (tmpdir string, cleanup func()) {
	tmpdir, err := ioutil.TempDir("", "dsl_test.")
	if err != nil {
		panic(err)
	}
	cleanup = func() {
		err := os.RemoveAll(tmpdir)
		if err != nil {
			panic(err)
		}
	}
	return
}

func mkfile(tmpdir string, basename string, data string) string {
	fn := path.Join(tmpdir, basename)
	err := ioutil.WriteFile(fn, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
	return fn
}
