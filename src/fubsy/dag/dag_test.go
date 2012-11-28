package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
)

type stubnode struct {
	nodebase
}

func (self *stubnode) Equal(other_ Node) bool {
	other, ok := other_.(*stubnode)
	return ok && self.name == other.name
}

func makestubnode(dag *DAG, name string) *stubnode {
	node := dag.lookup(name)
	if node == nil {
		node := &stubnode{
			nodebase: makenodebase(dag, -1, name),
		}
		node.id = dag.addNode(node)
		return node
	}
	return node.(*stubnode)
}

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.lookup("foo")
	assert.Nil(t, outnode)

	innode := &stubnode{nodebase: makenodebase(dag, -1, "foo")}
	id := dag.addNode(innode)
	assert.Equal(t, 0, id)
	assert.True(t, innode == dag.nodes[0].(*stubnode))

	outnode = dag.lookup("foo")
	assert.True(t, outnode.(*stubnode) == innode)

	assert.Nil(t, dag.lookup("bar"))
}

func Test_DAG_FindFinalTargets(t *testing.T) {
	// graph:
	//   node3: {node2, node1}
	//   node2: {node1}
	//   node0: {node2}
	// thus final targets = {node3, node0}
	// original sources = {node1}
	dag := NewDAG()
	node0 := makestubnode(dag, "node0")
	node1 := makestubnode(dag, "node1")
	node2 := makestubnode(dag, "node2")
	node3 := makestubnode(dag, "node3")
	node3.AddParent(node2)
	node3.AddParent(node1)
	node2.AddParent(node1)
	node0.AddParent(node1)

	targets := (*bit.Set)(dag.FindFinalTargets())
	assert.Equal(t, "{0, 3}", targets.String())
}
