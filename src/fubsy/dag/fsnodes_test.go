// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_MakeFileNode(t *testing.T) {
	dag := NewDAG()
	node0 := MakeFileNode(dag, "foo")
	assert.Equal(t, node0.name, "foo")
	assert.True(t, dag.nodes[0] == node0)
	assert.True(t, dag.index["foo"] == 0)

	node1 := MakeFileNode(dag, "bar")
	assert.Equal(t, node1.name, "bar")
	assert.True(t, dag.nodes[1] == node1)
	assert.True(t, dag.index["bar"] == 1)

	node0b := MakeFileNode(dag, "foo")
	assert.True(t, node0 == node0b)
}

func Test_FileNode_string(t *testing.T) {
	node := &FileNode{nodebase: nodebase{name: "foo/bar/baz"}}
	assert.Equal(t, "foo/bar/baz", node.Name())
	assert.Equal(t, "foo/bar/baz", node.String())
}

func Test_FileNode_parents(t *testing.T) {
	dag := NewDAG()
	node := MakeFileNode(dag, "foo/bar/qux")
	expect := []string{}
	assertParents(t, expect, dag, node)

	// add a single parent in isolation
	p0 := MakeFileNode(dag, "bong")
	expect = []string{"bong"}
	dag.addParent(node, p0)
	assertParents(t, expect, dag, node)

	// add a couple more
	p1 := MakeFileNode(dag, "blorp")
	p2 := MakeFileNode(dag, "meep")
	dag.addParent(node, p1)
	dag.addParent(node, p2)
	expect = append(expect, "blorp", "meep")
	assertParents(t, expect, dag, node)

	// ensure that duplicates are not re-added
	dag.addParent(node, p2)
	dag.addParent(node, p0)
	assertParents(t, expect, dag, node)
}

func Test_FileNode_parents_order(t *testing.T) {
	dag := NewDAG()
	node := MakeFileNode(dag, "foo")

	// test that AddParent() preserves order (highly unlikely that
	// hash order would preserve the sequence of 100 names by
	// coincidence!)
	expect := make([]string, 100)
	var name string
	for i := 0; i < 100; i++ {
		name = fmt.Sprintf("file%02d", i)
		expect[i] = name
		dag.addParent(node, MakeFileNode(dag, name))
	}
	assertParents(t, expect, dag, node)

	// More specifically, Parents() returns nodes ordered by node ID,
	// *not* by the order in which AddParent() was called (a
	// distinction that escapes the above loop). This is an
	// implementation detail of using a bitset; the important thing is
	// that the order of Parents() is consistent, deterministic,
	// non-arbitrary, and sensible to a human reader -- i.e. not
	// random and not hash order. Asserting that it's ordered by node
	// ID is a sanity check of the implementation, not part of the
	// interface.
	p1 := MakeFileNode(dag, "p1")
	p2 := MakeFileNode(dag, "p2")
	dag.addParent(node, p2)
	dag.addParent(node, p1)
	expect = append(expect, "p1", "p2")
	assertParents(t, expect, dag, node)
}

func Test_FileNode_action(t *testing.T) {
	dag := NewDAG()
	node := MakeFileNode(dag, "foo")

	action := &CommandAction{raw: types.FuString("ls -l")}
	node.SetAction(action)
	assert.Equal(t, action, node.Action())
}

func Test_FileNode_Exists(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.TouchFiles("foo.txt", "a/a/a/a/foo.txt", "a/b/unreadable")

	makeUnreadable("a/b")
	defer makeReadable("a/b")

	dag := NewDAG()
	tests := []struct {
		name   string
		exists bool
		err    string
	}{
		{"foo.txt", true, ""},
		{"a/a/a", false, "stat a/a/a: is a directory, not a regular file"},
		{"a/a/a/bogus", false, ""},
		{"a/a/a/a/foo.txt", true, ""},
		{"a/b/unreadable", false, "stat a/b/unreadable: permission denied"},
	}

	for _, test := range tests {
		node := MakeFileNode(dag, test.name)
		exists, err := node.Exists()
		if test.err != "" {
			assert.NotNil(t, err)
			assert.Equal(t, test.err, err.Error())
		}
		if test.exists && !exists {
			t.Errorf("%v: expected Exists() true, got false", node)
		} else if !test.exists && exists {
			t.Errorf("%v: expected Exists() false, got true", node)
		}
	}
}

func Benchmark_FileNode_AddParent(b *testing.B) {
	b.StopTimer()
	dag := NewDAG()
	nodes := make([]*FileNode, b.N)
	for i := range nodes {
		nodes[i] = MakeFileNode(dag, fmt.Sprintf("file%04d", i))
	}
	b.StartTimer()

	node := MakeFileNode(dag, "bop")
	for _, pnode := range nodes {
		dag.addParent(node, pnode)
	}
}

func Test_GlobNode_basics(t *testing.T) {
	dag := NewDAG()
	glob0 := types.NewFileFinder([]string{"**/*.java"})
	glob1 := types.NewFileFinder([]string{"doc/*/*.html"})
	glob2, err := glob0.Add(glob1) // it's a FuFinderList
	assert.Nil(t, err)

	node0 := MakeGlobNode(dag, glob0)
	node1 := MakeGlobNode(dag, glob1)
	node2 := MakeGlobNode(dag, glob2)

	// correctly reuse existing entries
	assert.Equal(t, dag.nodes[0], MakeGlobNode(dag, glob0))
	var obj types.FuObject = glob0
	assert.Equal(t, dag.nodes[0], MakeGlobNode(dag, obj))

	assert.Equal(t, "<**/*.java>", node0.String())
	assert.Equal(t, "<doc/*/*.html>", node1.String())
	assert.Equal(t, "<**/*.java> + <doc/*/*.html>", node2.String())

	assert.True(t, node0.Equal(node0))
	assert.False(t, node0.Equal(node1))
	assert.False(t, node0.Equal(node2))

	glob2b, err := glob0.Add(glob1)
	assert.Nil(t, err)
	assert.True(t, glob2b.Equal(glob2))
	node2b := MakeGlobNode(dag, glob2b)
	assert.True(t, node2b.Equal(node2))
}

func Test_GlobNode_Expand(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.TouchFiles(
		"src/util.c",
		"src/util.h",
		"src/util-test.c",
		"doc/README.txt",
		"main.c",
	)
	dag := NewDAG()
	node0 := MakeGlobNode(dag, types.NewFileFinder([]string{"*.c", "**/*.h"}))
	node1 := MakeGlobNode(dag, types.NewFileFinder([]string{"**/*.java"}))
	_ = node1

	expnodes, err := node0.Expand(dag, types.NewValueMap())
	assert.Nil(t, err)
	assert.Equal(t, 2, len(expnodes))
	assert.Equal(t, "main.c", expnodes[0].(*FileNode).name)
	assert.Equal(t, "src/util.h", expnodes[1].(*FileNode).name)
}

func assertParents(t *testing.T, expect []string, dag *DAG, node Node) {
	id := dag.lookupId(node)
	actual := dag.parentNodes(id)
	actualnames := make([]string, len(actual))
	for i, node := range actual {
		actualnames[i] = node.Name()
	}
	assert.Equal(t, expect, actualnames)
}

func makeUnreadable(name string) {
	chmodMask(name, ^os.ModePerm, 0)
}

func makeReadable(name string) {
	chmodMask(name, 0, 0700)
}

func chmodMask(name string, andmask, ormask os.FileMode) {
	// hmmm: does this work on windows?
	info, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	mode := info.Mode()&andmask | ormask
	err = os.Chmod(name, mode)
	if err != nil {
		panic(err)
	}
}
