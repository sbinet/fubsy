package fubsy

import (
	"testing"
	"os"
	"io/ioutil"
	"path"
)

func TestRootNode_Equal(t *testing.T) {
	node1 := RootNode{}
	node2 := RootNode{}
	if !node1.Equal(node1) {
		t.Error("root node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty root nodes not equal")
	}
	node1.elements = []ASTNode {ListNode{}}
	if node1.Equal(node2) {
		t.Error("non-empty root node equals empty root node")
	}
	node2.elements = []ASTNode {ListNode{}}
	if !node1.Equal(node2) {
		t.Error("root nodes with one child each not equal")
	}

	other := ListNode{}
	if node1.Equal(other) {
		t.Error("nodes of different type are equal")
	}
}

func TestListNode_Equal(t *testing.T) {
	node1 := ListNode{}
	node2 := ListNode{}
	if !node1.Equal(node1) {
		t.Error("list node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty list nodes not equal")
	}
	node1.values = []string {"bop"}
	if !node1.Equal(node1) {
		t.Error("non-empty list node not equal to itself")
	}
	if node1.Equal(node2) {
		t.Error("non-empty list node equal to empty list node")
	}
	node2.values = []string {"pop"}
	if node1.Equal(node2) {
		t.Error("list node equal to list node with different element")
	}
	node2.values[0] = "bop"
	if !node1.Equal(node2) {
		t.Error("equivalent list nodes not equal")
	}
	node1.values = append(node1.values, "boo")
	if node1.Equal(node2) {
		t.Error("list node equal to list node with different length")
	}
}

func TestInlineNode_Equal(t *testing.T) {
	node1 := InlineNode{}
	node2 := InlineNode{}
	if !node1.Equal(node1) {
		t.Error("InlineNode not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty InlineNodes not equal")
	}
	node1.lang = "foo"
	node2.lang = "bar"
	if node1.Equal(node2) {
		t.Error("InlineNodes equal despite different lang")
	}
	node2.lang = "foo"
	if !node1.Equal(node2) {
		t.Error("InlineNodes not equal")
	}
	node1.content = "hello\nworld\n"
	node2.content = "hello\nworld"
	if node1.Equal(node2) {
		t.Error("InlineNodes equal despite different content")
	}
	node2.content += "\n"
	if !node1.Equal(node2) {
		t.Error("InlineNodes not equal")
	}
}

func TestParse_valid_1(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// dead simple: a single top-level element
	fn := mkfile(tmpdir, "valid_1.fubsy", "main {\n[\"meep\"]\n}\n")
	//fn := mkfile(tmpdir, "valid_1.fubsy", "main{[\"foo\"][\"bar\"]}")

	expect := RootNode{elements: []ASTNode {
			PhaseNode{name: "main", statements: []ASTNode {
					ListNode {values: []string {"meep"}}}}}}
	ast, err := Parse(fn)
	assertNoError(t, err)
	assertASTEquals(t, &expect, ast)
}

func TestParse_valid_2(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// sequence of top-level elements
	fn := mkfile(
		tmpdir,
		"valid_2.fubsy",
		"main {\n[\"boo\"]\n}\n" +
		"plugin foo {{{o'malley & friends\n}}}\n" +
		"blob { [\"meep\"] }")
	ast, err := Parse(fn)
	assertNoError(t, err)

	expect := RootNode{elements: []ASTNode {
			PhaseNode{
				name: "main",
				statements: []ASTNode {ListNode{values: []string {"boo"}}}},
			InlineNode{
				lang: "foo", content: "o'malley & friends\n"},
			PhaseNode{
				name: "blob",
				statements: []ASTNode {ListNode{values: []string {"meep"}}}},
	}}
	assertASTEquals(t, &expect, ast)
}

func TestParse_invalid_1(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: no closing rbracket
	fn := mkfile(tmpdir, "invalid_1.fubsy", "main{  [\n\"borf\"\n }")
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near })"
	assertError(t, expect, err)
}

func TestParse_invalid_2(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: bad token
	fn := mkfile(tmpdir, "invalid_2.fubsy", "main\n{[\n *&! \"whizz\"]\n}")
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near *&!)"
	assertError(t, expect, err)
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
