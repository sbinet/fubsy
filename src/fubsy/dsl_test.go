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

func TestParse_valid(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	fn := mkfile(tmpdir, "valid.fubsy", "  [\n\"hey ${there}\"    ]\n ")

	expect := RootNode{elements: []ASTNode {ListNode{values: []string {"hey ${there}"}}}}
	ast_, err := Parse(fn)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if ast, ok := ast_.(*RootNode); ok {
		checkASTEquals(t, &expect, ast)
	} else {
		t.Fatalf("expected ast_ to be RootNode, not %v", ast_)
	}
}

func TestParse_invalid_1(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: no closing rbracket
	fn := mkfile(tmpdir, "invalid_1.fubsy", "  [\n\"borf\"\n ")
	_, err := Parse(fn)
	expect := fn + ":2: syntax error (near \"borf\")"
	assertError(t, expect, err)
}

func TestParse_invalid_2(t *testing.T) {
	tmpdir, cleanup := mktemp()
	defer cleanup()

	// invalid: bad token
	fn := mkfile(tmpdir, "invalid_2.fubsy", "\n [\n xx \"whizz\"]\n")
	_, err := Parse(fn)
	expect := fn + ":3: syntax error (near xx)"
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

func assertError(t *testing.T, expect string, actual error) {
	if actual == nil {
		t.Fatal("expected error, but got nil")
	}
	if actual.Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual.Error())
	}
}
