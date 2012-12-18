// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fmt"
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

func Test_FileNode_buildrule(t *testing.T) {
	// this really tests the implementation in nodebase (node.go)
	dag := NewDAG()
	node := MakeFileNode(dag, "foo")
	assert.Nil(t, node.BuildRule())

	rule := &stubrule{targets: []Node{node}}
	node.SetBuildRule(rule)
	assert.Equal(t, rule, node.BuildRule())
}

func Test_FileNode_Expand(t *testing.T) {
	ns := types.NewValueMap()
	node := newFileNode("foobar")
	xnode, err := node.Expand(ns)
	assert.Nil(t, err)
	assert.Equal(t, node, xnode)
}

func Test_FileNode_Exists(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.TouchFiles("foo.txt", "a/a/a/a/foo.txt", "a/b/unreadable")

	testutils.ChmodNoAccess("a/b")
	defer testutils.ChmodOwnerAll("a/b")

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

func assertParents(t *testing.T, expect []string, dag *DAG, node Node) {
	id := dag.lookupId(node)
	actual := dag.parentNodes(id)
	actualnames := make([]string, len(actual))
	for i, node := range actual {
		actualnames[i] = node.Name()
	}
	assert.Equal(t, expect, actualnames)
}
