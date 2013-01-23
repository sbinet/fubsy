// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fmt"
	"hash/fnv"
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
	dag.AddParent(node, p0)
	assertParents(t, expect, dag, node)

	// add a couple more
	p1 := MakeFileNode(dag, "blorp")
	p2 := MakeFileNode(dag, "meep")
	dag.AddParent(node, p1)
	dag.AddParent(node, p2)
	expect = append(expect, "blorp", "meep")
	assertParents(t, expect, dag, node)

	// ensure that duplicates are not re-added
	dag.AddParent(node, p2)
	dag.AddParent(node, p0)
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
		dag.AddParent(node, MakeFileNode(dag, name))
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
	dag.AddParent(node, p2)
	dag.AddParent(node, p1)
	expect = append(expect, "p1", "p2")
	assertParents(t, expect, dag, node)
}

func Test_FileNode_buildrule(t *testing.T) {
	// this really tests the implementation in nodebase (node.go)
	dag := NewDAG()
	node := MakeFileNode(dag, "foo")
	assert.Nil(t, node.BuildRule())

	rule := &StubRule{targets: []Node{node}}
	node.SetBuildRule(rule)
	assert.Equal(t, rule, node.BuildRule())
}

func Test_FileNode_Expand(t *testing.T) {
	ns := types.NewValueMap()
	node := newFileNode("foobar")
	xnode, err := node.ActionExpand(ns, nil)
	assert.Nil(t, err)
	assert.Equal(t, node, xnode)

	err = node.NodeExpand(ns)
	assert.Nil(t, err)
	assert.Equal(t, "foobar", node.Name())

	// test that ActionExpand() follows variable references
	node = newFileNode("$foo$bar")
	xnode, err = node.ActionExpand(ns, nil)
	assert.Equal(t, "undefined variable 'foo' in string", err.Error())

	// make it so "$foo$bar" expands to "$foo", and ensure that
	// expansion stops there
	// XXX argh: currently this expands to "'$'foo": hmmmmm
	// ns.Assign("foo", types.FuString("$"))
	// ns.Assign("bar", types.FuString("foo"))
	// xnode, err = node.ActionExpand(ns, nil)
	// assert.Nil(t, err)
	// assert.Equal(t, "$foo", xnode.String())
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

func Test_FileNode_Signature(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.Mkdirs("d1", "d2")
	testutils.Mkfile("d1", "empty", "")
	testutils.Mkfile("d2", "stuff", "foo\n")

	node1 := newFileNode("d1/empty")
	node2 := newFileNode("d2/stuff")
	node3 := newFileNode("nonexistent")

	expect := []byte{}
	hash := fnv.New64a()
	assert.Equal(t, 8, hash.Size())
	expect = hash.Sum(expect)

	sig, err := node1.Signature()
	assert.Nil(t, err)
	assert.Equal(t, expect, sig)

	hash.Write([]byte{'f', 'o', 'o', '\n'})
	expect = expect[:0]
	expect = hash.Sum(expect)
	sig, err = node2.Signature()
	assert.Nil(t, err)
	assert.Equal(t, expect, sig)

	// make sure it's cached, i.e. changes to the file are not seen by
	// the same FileNode object in the same process
	testutils.Mkfile("d2", "stuff", "fooo\n")
	sig, err = node2.Signature()
	assert.Nil(t, err)
	assert.Equal(t, expect, sig)

	// in fact, even if the file disappears, we still have its signature
	err = os.Remove("d2/stuff")
	if err != nil {
		panic(err)
	}
	sig, err = node2.Signature()
	assert.Nil(t, err)
	assert.Equal(t, expect, sig)

	sig, err = node3.Signature()
	assert.NotNil(t, err)
	assert.Equal(t, "open nonexistent: no such file or directory", err.Error())
}

func Test_FileNode_Changed(t *testing.T) {
	// this is really a test of Signature() + Changed() together, because
	// Changed() itself is so trivial that testing it is no challenge
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.Mkfile(".", "stuff.txt", "blah blah blah\n")
	node := newFileNode("stuff.txt")
	osig, err := node.Signature()
	assert.Nil(t, err)

	// construct a new FileNode so the cache is lost
	node = newFileNode("stuff.txt")
	nsig, err := node.Signature()
	assert.Nil(t, err)
	assert.False(t, node.Changed(osig, nsig))

	// modify the file and repeat
	testutils.Mkfile(".", "stuff.txt", "blah blah blah\nblah")
	node = newFileNode("stuff.txt")
	nsig, err = node.Signature()
	assert.Nil(t, err)
	assert.True(t, node.Changed(osig, nsig))
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
		dag.AddParent(node, pnode)
	}
}

func assertParents(t *testing.T, expect []string, dag *DAG, node Node) {
	actual := dag.ParentNodes(node)
	actualnames := make([]string, len(actual))
	for i, node := range actual {
		actualnames[i] = node.Name()
	}
	assert.Equal(t, expect, actualnames)
}
