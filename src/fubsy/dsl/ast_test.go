// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_NewASTRoot(t *testing.T) {
	children := []ASTNode{
		NewASTName("foo"),
		NewASTName("bar"),
	}
	root := NewASTRoot(children)

	assert.Equal(t, children, root.Children())

	children = children[:0]
	root = NewASTRoot(children)
	assert.Equal(t, children, root.Children())
}

func Test_ASTRoot_Equal(t *testing.T) {
	node1 := &ASTRoot{}
	node2 := &ASTRoot{}
	if !node1.Equal(node1) {
		t.Error("root node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty root nodes not equal")
	}
	node1.children = []ASTNode{&ASTString{}}
	if node1.Equal(node2) {
		t.Error("non-empty root node equals empty root node")
	}
	node2.children = []ASTNode{&ASTString{}}
	if !node1.Equal(node2) {
		t.Error("root nodes with one child each not equal")
	}

	other := &ASTString{}
	if node1.Equal(other) {
		t.Error("nodes of different type are equal")
	}
}

func Test_ASTRoot_FindImports(t *testing.T) {
	root := &ASTRoot{
		children: []ASTNode{
			&ASTPhase{},
			&ASTPhase{},
			&ASTImport{plugin: []string{"ding"}},
			&ASTInline{},
			&ASTImport{plugin: []string{"meep", "beep"}},
		}}
	expect := [][]string{{"ding"}, {"meep", "beep"}}
	actual := root.FindImports()
	assert.True(t, reflect.DeepEqual(expect, actual),
		"expected\n%v\nbut got\n%v", expect, actual)
}

func Test_ASTRoot_Phase(t *testing.T) {
	root := &ASTRoot{
		children: []ASTNode{
			&ASTImport{},
			&ASTPhase{name: "meep"},
			&ASTInline{},
			&ASTPhase{name: "meep"}, // duplicate is invisible
			&ASTPhase{name: "bong"},
		}}
	var expect *ASTPhase
	var actual *ASTPhase
	actual = root.FindPhase("main")
	assert.Nil(t, actual)

	expect = root.children[1].(*ASTPhase)
	actual = root.FindPhase("meep")
	assert.True(t, expect == actual,
		"expected %p (%v)\nbut got %p (%v)",
		expect, expect, actual, actual)

	expect = root.children[4].(*ASTPhase)
	actual = root.FindPhase("bong")
	assert.True(t, expect == actual,
		"expected %p\n%#v\nbut got %p\n%#v",
		expect, expect, actual, actual)
}

func Test_ASTPhase_Equal(t *testing.T) {
	phase1 := &ASTPhase{name: "foo"}
	phase2 := &ASTPhase{name: "foo"}
	assert.True(t, phase1.Equal(phase2), "phase nodes not equal")
}

func Test_ASTFileFinder_Equal(t *testing.T) {
	node1 := &ASTFileFinder{}
	node2 := &ASTFileFinder{}
	assert.True(t, node1.Equal(node1),
		"list node not equal to itself")
	assert.True(t, node1.Equal(node2),
		"empty list nodes not equal")
	node1.patterns = []string{"bop"}
	assert.True(t, node1.Equal(node1),
		"non-empty list node not equal to itself")
	assert.False(t, node1.Equal(node2),
		"non-empty list node equal to empty list node")
	node2.patterns = []string{"pop"}
	assert.False(t, node1.Equal(node2),
		"list node equal to list node with different element")
	node2.patterns[0] = "bop"
	assert.True(t, node1.Equal(node2),
		"equivalent list nodes not equal")
	node1.patterns = append(node1.patterns, "boo")
	assert.False(t, node1.Equal(node2),
		"list node equal to list node with different length")
}

func Test_ASTInline_Equal(t *testing.T) {
	node1 := &ASTInline{}
	node2 := &ASTInline{}
	assert.True(t, node1.Equal(node1),
		"ASTInline not equal to itself")
	assert.True(t, node1.Equal(node2),
		"empty ASTInlines not equal")

	node1.lang = "foo"
	node2.lang = "bar"
	assert.False(t, node1.Equal(node2),
		"ASTInlines equal despite different lang")

	node2.lang = "foo"
	assert.True(t, node1.Equal(node2),
		"ASTInlines not equal")

	node1.content = "hello\nworld\n"
	node2.content = "hello\nworld"
	assert.False(t, node1.Equal(node2),
		"ASTInlines equal despite different content")

	node2.content += "\n"
	assert.True(t, node1.Equal(node2),
		"ASTInlines not equal")
}

func Test_ASTInline_Dump(t *testing.T) {
	tests := []struct {
		input  string
		indent string
		expect string
	}{
		{"", "", "ASTInline[foo] {{{\n}}}\n"},
		{"", " ", " ASTInline[foo] {{{\n }}}\n"},
		{"foobar", "  ", "  ASTInline[foo] {{{\n    foobar\n  }}}\n"},
		{"foobar\n", "", "ASTInline[foo] {{{\n  foobar\n  \n}}}\n"},
		{"hello\nworld", "", "ASTInline[foo] {{{\n  hello\n  world\n}}}\n"},
		{"\nhello\nworld", ".", ".ASTInline[foo] {{{\n.  \n.  hello\n.  world\n.}}}\n"},
		{"\nhello\nworld\n", "", "ASTInline[foo] {{{\n  \n  hello\n  world\n  \n}}}\n"},
		{"\nhello\nworld\n", "!", "!ASTInline[foo] {{{\n!  \n!  hello\n!  world\n!  \n!}}}\n"},
		{"hello\n  world", "%%", "%%ASTInline[foo] {{{\n%%  hello\n%%    world\n%%}}}\n"},
		{"hello\n  world\n", "", "ASTInline[foo] {{{\n  hello\n    world\n  \n}}}\n"},
	}

	node := &ASTInline{lang: "foo"}
	for i, test := range tests {
		var buf bytes.Buffer
		node.content = test.input
		node.Dump(&buf, test.indent)
		actual := buf.String()
		if test.expect != actual {
			t.Errorf("ASTInline.Dump() %d: expected\n%#v\nbut got\n%#v",
				i, test.expect, actual)
		}
	}
}

func Test_ASTName_Equal_location(t *testing.T) {
	name1 := &ASTName{name: "foo"}
	name2 := &ASTName{name: "foo"}
	assert.True(t, name1.Equal(name2),
		"obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int{}}
	name1.location = FileLocation{fileinfo, 0, 5}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location set")

	name2.location = FileLocation{fileinfo, 5, 7}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to different values")

	name2.location = FileLocation{fileinfo, 0, 5}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to equal values")
}

func Test_ASTList_Equal(t *testing.T) {
	val1 := NewASTName("a")
	val2 := NewASTName("b", NewStubLocation("loc1"))
	val3 := NewASTName("b", NewStubLocation("loc2"))
	list1 := &ASTList{elements: []ASTExpression{val1, val2}}
	list2 := &ASTList{elements: []ASTExpression{val1, val3}}
	list3 := &ASTList{elements: []ASTExpression{val3, val1}}

	assert.True(t, list1.Equal(list2))
	assert.False(t, list1.Equal(list3))
}

func Test_ASTFunctionCall_Equal_location(t *testing.T) {
	// location is irrelevant to comparison
	fcall1 := &ASTFunctionCall{
		function: &ASTName{name: "foo"},
		args:     []ASTExpression{&ASTString{value: "bar"}}}
	fcall2 := &ASTFunctionCall{
		function: &ASTName{name: "foo"},
		args:     []ASTExpression{&ASTString{value: "bar"}}}
	assert.True(t, fcall1.Equal(fcall2),
		"obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int{}}
	fcall1.location = FileLocation{fileinfo, 3, 18}
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall1 has location but fcall2 does not")

	fcall2.location = fcall1.location
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location is a copy of fcall1's")

	fcall2.location = FileLocation{fileinfo, 5, 41}
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location different from fcall1's")
}
