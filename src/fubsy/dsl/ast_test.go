// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"testing"
	"reflect"
	"bytes"
	"github.com/stretchrcom/testify/assert"
)

func Test_ASTRoot_Equal(t *testing.T) {
	node1 := &ASTRoot{}
	node2 := &ASTRoot{}
	if !node1.Equal(node1) {
		t.Error("root node not equal to itself")
	}
	if !node1.Equal(node2) {
		t.Error("empty root nodes not equal")
	}
	node1.children = []ASTNode {&ASTString{}}
	if node1.Equal(node2) {
		t.Error("non-empty root node equals empty root node")
	}
	node2.children = []ASTNode {&ASTString{}}
	if !node1.Equal(node2) {
		t.Error("root nodes with one child each not equal")
	}

	other := &ASTString{}
	if node1.Equal(other) {
		t.Error("nodes of different type are equal")
	}
}

func Test_ASTRoot_ListPlugins(t *testing.T) {
	root := &ASTRoot{
		children: []ASTNode {
			&ASTPhase{},
			&ASTPhase{},
			&ASTImport{plugin: []string {"ding"}},
			&ASTInline{},
			&ASTImport{plugin: []string {"meep", "beep"}},
	}}
	expect := [][]string {{"ding"}, {"meep", "beep"}}
	actual := root.ListPlugins()
	assert.True(t, reflect.DeepEqual(expect, actual),
	 	"expected\n%v\nbut got\n%v", expect, actual)
}

func Test_ASTRoot_Phase(t *testing.T) {
	root := &ASTRoot{
		children: []ASTNode {
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

func Test_ASTFileList_Equal(t *testing.T) {
	node1 := &ASTFileList{}
	node2 := &ASTFileList{}
	assert.True(t, node1.Equal(node1),
		"list node not equal to itself")
	assert.True(t, node1.Equal(node2),
		"empty list nodes not equal")
	node1.patterns = []string {"bop"}
	assert.True(t, node1.Equal(node1),
		"non-empty list node not equal to itself")
	assert.False(t, node1.Equal(node2),
		"non-empty list node equal to empty list node")
	node2.patterns = []string {"pop"}
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
	node := &ASTInline{lang: "foo"}
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
	name1 := &ASTName{name: "foo"}
	name2 := &ASTName{name: "foo"}
	assert.True(t, name1.Equal(name2),
		"obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int {}}
	name1.location = Location{fileinfo, 0, 5}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location set")

	name2.location = Location{fileinfo, 5, 7}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to different values")

	name2.location = Location{fileinfo, 0, 5}
	assert.True(t, name1.Equal(name2),
		"equality fails with name1.location and name2.location set to equal values")
}

func Test_ASTFunctionCall_Equal_location(t *testing.T) {
	// location is irrelevant to comparison
	fcall1 := &ASTFunctionCall{
		function: &ASTName{name: "foo"},
		args: []ASTExpression {&ASTString{value: "bar"}}}
	fcall2 := &ASTFunctionCall{
		function: &ASTName{name: "foo"},
		args: []ASTExpression {&ASTString{value: "bar"}}}
	assert.True(t, fcall1.Equal(fcall2),
		"obvious equality fails")

	fileinfo := &fileinfo{"foo.txt", []int {}}
	fcall1.location = Location{fileinfo, 3, 18}
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall1 has location but fcall2 does not")

	fcall2.location = fcall1.location
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location is a copy of fcall1's")

	fcall2.location = Location{fileinfo, 5, 41}
	assert.True(t, fcall1.Equal(fcall2),
		"equality fails when fcall2's location different from fcall1's")
}

func assertASTDump(t *testing.T, expect string, node ASTNode) {
	var buf bytes.Buffer
	node.Dump(&buf, "")
	actual := buf.String()
	assert.Equal(t, expect, actual, "AST dump")
}
