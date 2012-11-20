package dag

import (
	"testing"
	"fmt"
	"github.com/stretchrcom/testify/assert"
)

func Test_makeFileNode(t *testing.T) {
	dag := NewDAG()
	node0 := makeFileNode(dag, "foo")
	assert.Equal(t, node0.name, "foo")
	assert.Equal(t, node0.id, 0)
	assert.True(t, dag.nodes[0] == node0)
	assert.True(t, dag.index["foo"] == 0)

	node1 := makeFileNode(dag, "bar")
	assert.Equal(t, node1.name, "bar")
	assert.Equal(t, node1.Id(), 1)
	assert.True(t, dag.nodes[1] == node1)
	assert.True(t, dag.index["bar"] == 1)

	node0b := makeFileNode(dag, "foo")
	assert.True(t, node0 == node0b)
}

func Test_FileNode_parents(t *testing.T) {
	dag := NewDAG()
	node := makeFileNode(dag, "foo/bar/qux")
	expect := []string {}
	assertParents(t, expect, node)

	// add a single parent in isolation
	p0 := makeFileNode(dag, "bong")
	expect = []string {"bong"}
	node.AddParent(p0)
	assertParents(t, expect, node)

	// test that AddParent() preserves order (highly unlikely that
	// hash order would preserve the sequence of 100 names by
	// coincidence!)
	var name string
	for i := 0; i < 100; i++ {
		name = fmt.Sprintf("file%02d", i)
		node.AddParent(makeFileNode(dag, name))
		expect = append(expect, name)
	}
	assertParents(t, expect, node)

	// ensure that duplicates are not re-added
	node.AddParent(node.parents[53])
	node.AddParent(node.parents[17])
	node.AddParent(node.parents[75])
	assertParents(t, expect, node)

	// again, but with new node objects (not reused)
	node.AddParent(makeFileNode(dag, "file63"))
	node.AddParent(makeFileNode(dag, "file19"))
	assertParents(t, expect, node)
}

func Benchmark_FileNode_AddParent(b *testing.B) {
	b.StopTimer()
	dag := NewDAG()
	nodes := make([]*FileNode, b.N)
	for i := range nodes {
		nodes[i] = makeFileNode(dag, fmt.Sprintf("file%04d", i))
	}
	b.StartTimer()

	node := makeFileNode(dag, "bop")
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
