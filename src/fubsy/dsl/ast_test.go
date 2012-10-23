package dsl

import (
	"testing"
	"bytes"
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

func TestInlineNode_Dump(t *testing.T) {
	node := InlineNode{lang: "foo"}
	assertASTDump(t, "InlineNode[foo] {{{}}}\n", node)

	node.content = "foobar"
	assertASTDump(t, "InlineNode[foo] {{{foobar}}}\n", node)

	node.content = "foobar\n"
	assertASTDump(t, "InlineNode[foo] {{{foobar\n}}}\n", node)

	node.content = "hello\nworld"
	assertASTDump(t, "InlineNode[foo] {{{hello\n  world}}}\n", node)

	node.content = "\nhello\nworld"
	assertASTDump(t, "InlineNode[foo] {{{\n  hello\n  world}}}\n", node)

	node.content = "\nhello\nworld\n"
	assertASTDump(t, "InlineNode[foo] {{{\n  hello\n  world\n}}}\n", node)

	node.content = "hello\n  world"
	assertASTDump(t, "InlineNode[foo] {{{hello\n    world}}}\n", node)

	node.content = "hello\n  world\n"
	assertASTDump(t, "InlineNode[foo] {{{hello\n    world\n}}}\n", node)

}

func assertASTDump(t *testing.T, expect string, node ASTNode) {
	var buf bytes.Buffer
	node.Dump(&buf, "")
	actual := buf.String()
	if expect != actual {
		t.Errorf("AST dump: expected\n%s\nbut got\n%s", expect, actual)
	}
}
