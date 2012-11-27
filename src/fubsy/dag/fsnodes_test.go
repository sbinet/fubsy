package dag

import (
	"testing"
	"fmt"
	"github.com/stretchrcom/testify/assert"
)

func Test_MakeFileNode(t *testing.T) {
	dag := NewDAG()
	node0 := MakeFileNode(dag, "foo")
	assert.Equal(t, node0.name, "foo")
	assert.Equal(t, node0.id, 0)
	assert.True(t, dag.nodes[0] == node0)
	assert.True(t, dag.index["foo"] == 0)

	node1 := MakeFileNode(dag, "bar")
	assert.Equal(t, node1.name, "bar")
	assert.Equal(t, node1.Id(), 1)
	assert.True(t, dag.nodes[1] == node1)
	assert.True(t, dag.index["bar"] == 1)

	node0b := MakeFileNode(dag, "foo")
	assert.True(t, node0 == node0b)
}

func Test_FileNode_parents(t *testing.T) {
	dag := NewDAG()
	node := MakeFileNode(dag, "foo/bar/qux")
	expect := []string {}
	assertParents(t, expect, node)

	// add a single parent in isolation
	p0 := MakeFileNode(dag, "bong")
	expect = []string {"bong"}
	node.AddParent(p0)
	assertParents(t, expect, node)

	// add a couple more
	p1 := MakeFileNode(dag, "blorp")
	p2 := MakeFileNode(dag, "meep")
	node.AddParent(p1)
	node.AddParent(p2)
	expect = append(expect, "blorp", "meep")
	assertParents(t, expect, node)

	// ensure that duplicates are not re-added
	node.AddParent(p2)
	node.AddParent(p0)
	assertParents(t, expect, node)
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
		node.AddParent(MakeFileNode(dag, name))
	}
	assertParents(t, expect, node)

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
	node.AddParent(p2)
	node.AddParent(p1)
	expect = append(expect, "p1", "p2")
	assertParents(t, expect, node)
}

func Test_FileNode_action(t *testing.T) {
	dag := NewDAG()
	node := MakeFileNode(dag, "foo")

	action := &CommandAction{raw: "ls -l"}
	node.SetAction(action)
	assert.Equal(t, action, node.Action())
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
		node.AddParent(pnode)
	}
}

func assertParents(t *testing.T, expect []string, node Node) {
	actual := node.Parents()	// list of Node
	actualnames := make([]string, len(actual))
	for i, node := range actual {
		actualnames[i] = node.Name()
	}
	assert.Equal(t, expect, actualnames)
}
