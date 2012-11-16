package dsl

import (
	"testing"
	"reflect"
	"bytes"
)

func Test_ASTRoot_Equal(t *testing.T) {
	node1 := ASTRoot{}
	node2 := ASTRoot{}
	if !node1.Equal(node1) {
		t.Error("root node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty root nodes not equal")
	}
	node1.children = []ASTNode {ASTString{}}
	if node1.Equal(node2) {
		t.Error("non-empty root node equals empty root node")
	}
	node2.children = []ASTNode {ASTString{}}
	if !node1.Equal(node2) {
		t.Error("root nodes with one child each not equal")
	}

	other := ASTString{}
	if node1.Equal(other) {
		t.Error("nodes of different type are equal")
	}
}

func Test_ASTRoot_ListPlugins(t *testing.T) {
	root := ASTRoot{
		children: []ASTNode {
			ASTPhase{},
			ASTPhase{},
			ASTImport{plugin: []string {"ding"}},
			ASTInline{},
			ASTImport{plugin: []string {"meep", "beep"}},
	}}
	expect := [][]string {{"ding"}, {"meep", "beep"}}
	actual := root.ListPlugins()
	assertTrue(t, reflect.DeepEqual(expect, actual),
		"expected\n%v\nbut got\n%v", expect, actual)
}

func Test_ASTRoot_Phase(t *testing.T) {
	root := ASTRoot{
		children: []ASTNode {
			ASTImport{},
			ASTPhase{name: "meep"},
			ASTInline{},
			ASTPhase{name: "meep"}, // duplicate is invisible
			ASTPhase{name: "bong"},
	}}
	var expect ASTPhase
	var actual *ASTPhase
	//expect = nil
	actual = root.FindPhase("main")
	assertTrue(t, nil == actual, "expected nil, but got %v", actual)

	// hmmm: would be nice to compare pointers to guarantee that we're
	// not copying structs, but I'm pretty sure we *do* copy structs
	// because we're not putting pointers into (say) root.children
	expect = root.children[1].(ASTPhase)
	actual = root.FindPhase("meep")
	assertTrue(t, expect.Equal(*actual), "expected\n%#v\nbut got\n%#v",
		expect, actual)

	expect = root.children[4].(ASTPhase)
	actual= root.FindPhase("bong")
	assertTrue(t, expect.Equal(*actual), "expected\n%#v\nbut got\n%#v",
		expect, actual)
}

func Test_ASTFileList_Equal(t *testing.T) {
	node1 := ASTFileList{}
	node2 := ASTFileList{}
	if !node1.Equal(node1) {
		t.Error("list node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty list nodes not equal")
	}
	node1.patterns = []string {"bop"}
	if !node1.Equal(node1) {
		t.Error("non-empty list node not equal to itself")
	}
	if node1.Equal(node2) {
		t.Error("non-empty list node equal to empty list node")
	}
	node2.patterns = []string {"pop"}
	if node1.Equal(node2) {
		t.Error("list node equal to list node with different element")
	}
	node2.patterns[0] = "bop"
	if !node1.Equal(node2) {
		t.Error("equivalent list nodes not equal")
	}
	node1.patterns = append(node1.patterns, "boo")
	if node1.Equal(node2) {
		t.Error("list node equal to list node with different length")
	}
}

func Test_ASTInline_Equal(t *testing.T) {
	node1 := ASTInline{}
	node2 := ASTInline{}
	if !node1.Equal(node1) {
		t.Error("ASTInline not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty ASTInlines not equal")
	}
	node1.lang = "foo"
	node2.lang = "bar"
	if node1.Equal(node2) {
		t.Error("ASTInlines equal despite different lang")
	}
	node2.lang = "foo"
	if !node1.Equal(node2) {
		t.Error("ASTInlines not equal")
	}
	node1.content = "hello\nworld\n"
	node2.content = "hello\nworld"
	if node1.Equal(node2) {
		t.Error("ASTInlines equal despite different content")
	}
	node2.content += "\n"
	if !node1.Equal(node2) {
		t.Error("ASTInlines not equal")
	}
}

func Test_ASTInline_Dump(t *testing.T) {
	node := ASTInline{lang: "foo"}
	assertASTDump(t, "ASTInline[foo] {{{}}}\n", node)

	node.content = "foobar"
	assertASTDump(t, "ASTInline[foo] {{{foobar}}}\n", node)

	node.content = "foobar\n"
	assertASTDump(t, "ASTInline[foo] {{{foobar\n}}}\n", node)

	node.content = "hello\nworld"
	assertASTDump(t, "ASTInline[foo] {{{hello\n  world}}}\n", node)

	node.content = "\nhello\nworld"
	assertASTDump(t, "ASTInline[foo] {{{\n  hello\n  world}}}\n", node)

	node.content = "\nhello\nworld\n"
	assertASTDump(t, "ASTInline[foo] {{{\n  hello\n  world\n}}}\n", node)

	node.content = "hello\n  world"
	assertASTDump(t, "ASTInline[foo] {{{hello\n    world}}}\n", node)

	node.content = "hello\n  world\n"
	assertASTDump(t, "ASTInline[foo] {{{hello\n    world\n}}}\n", node)

}

func Test_ASTName_Equal_location(t *testing.T) {
	name1 := ASTName{name: "foo"}
	name2 := ASTName{name: "foo"}
	assertTrue(t, name1.Equal(name2), "obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int {}}
	name1.location = location{fileinfo, 0, 5}
	assertTrue(t, name1.Equal(name2), "equality fails with name1.location set")

	name2.location = location{fileinfo, 5, 7}
	assertTrue(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to different values")

	name2.location = location{fileinfo, 0, 5}
	assertTrue(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to equal values")
}

func Test_ASTFunctionCall_Equal_location(t *testing.T) {
	// location is irrelevant to comparison
	fcall1 := ASTFunctionCall{
		function: ASTName{name: "foo"},
		args: []ASTExpression {ASTString{value: "bar"}}}
	fcall2 := ASTFunctionCall{
		function: ASTName{name: "foo"},
		args: []ASTExpression {ASTString{value: "bar"}}}
	assertTrue(t, fcall1.Equal(fcall2), "obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int {}}
	fcall1.location = location{fileinfo, 3, 18}
	assertTrue(t, fcall1.Equal(fcall2),
		"equality fails when fcall1 has location but fcall2 does not")

	fcall2.location = fcall1.location
	assertTrue(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location is a copy of fcall1's")

	fcall2.location = location{fileinfo, 5, 41}
	assertTrue(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location different from fcall1's")
}

func assertASTDump(t *testing.T, expect string, node ASTNode) {
	var buf bytes.Buffer
	node.Dump(&buf, "")
	actual := buf.String()
	if expect != actual {
		t.Errorf("AST dump: expected\n%s\nbut got\n%s", expect, actual)
	}
}
