package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
)

type stubnode struct {
	id int
	name string
}

func (self stubnode) Id() int {
	return self.id
}

func (self stubnode) Name() string {
	return self.name
}

func (self stubnode) Equal(other_ Node) bool {
	other, ok := other_.(stubnode)
	return ok && self.name == other.name
}

func (self stubnode) Parents() []Node {
	panic("stubnode.Parents() not implemented")
}

func (self stubnode) AddParent(node Node) {
	panic("stubnode.AddParent() not implemented")
}

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.lookup("foo")
	assert.Nil(t, outnode)

	// use a pointer so we can test that DAG returns the same object,
	// not a copy
	innode := &stubnode{-1, "foo"}
	id := dag.addNode(innode)
	assert.Equal(t, 0, id)
	assert.True(t, innode == dag.nodes[0].(*stubnode))

	outnode = dag.lookup("foo")
	assert.True(t, outnode.(*stubnode) == innode)

	assert.Nil(t, dag.lookup("bar"))
}
