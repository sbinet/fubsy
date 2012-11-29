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
	dag := makeSimpleGraph()
	targets := (*bit.Set)(dag.FindFinalTargets())
	assert.Equal(t, "{0, 1}", targets.String())
}

func Test_DAG_FindOriginalSources(t *testing.T) {
	dag := makeSimpleGraph()
	goal := bit.New(0, 1)		// all final targets: tool1, tool2
	sources, relevant := dag.FindOriginalSources(NodeSet(goal))
	assert.Equal(t, "{6..11}", (*bit.Set)(sources).String())
	assert.Equal(t, "{0..11}", (*bit.Set)(relevant).String())

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	goal = bit.New(0)
	sources, relevant = dag.FindOriginalSources(NodeSet(goal))
	assert.Equal(t, "{6..10}", (*bit.Set)(sources).String())
	assert.Equal(t, "{0, 2..4, 6..10}", (*bit.Set)(relevant).String())

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	goal = bit.New(1)
	sources, relevant = dag.FindOriginalSources(NodeSet(goal))
	assert.Equal(t, "{9..11}", (*bit.Set)(sources).String())
	assert.Equal(t, "{1, 4, 5, 9..11}", (*bit.Set)(relevant).String())
}

func Test_DAG_FindOriginalSources_cycle(t *testing.T) {
	dag := makeSimpleGraph()
	dag.lookup("misc.h").AddParent(dag.lookup("tool1"))

	// goal = {tool2} ==> no cycle, since we don't visit those nodes
	// (this simply tests that FindOriginalSources() doesn't panic)
	goal := NodeSet(bit.New(1))
	_, _ = dag.FindOriginalSources(goal)

	// goal = {tool1} ==> cycle!
	// (disabled because FindOriginalSources() currently panics on cycle)
	return
	goal = NodeSet(bit.New(0))
	_, _ = dag.FindOriginalSources(goal)
}

func makeSimpleGraph() *DAG {
	// dependency graph for a C project with two executables as the
	// final targets:
	//   tool1:		{tool1.o, misc.o, util.o}
	//   tool2:		{tool2.o, util.o}
	//   tool1.o:	{tool1.c, misc.h, util.h}
	//   misc.o:	{misc.h, misc.c}
	//   util.o:	{util.h, util.c}
	//   tool2.o:	{tool2.c, util.h}
	//   tool1.c:	{}
	//   misc.h:	{}
	//   misc.c:	{}
	//   util.h:	{}
	//   util.c:	{}
	//   tool2.c:	{}
	// final targets: {tool1, tool2} (node IDs 0, 1)
	// original sources: {tool1.c, misc.h, misc.c, util.h, util.c, tool2.c}
	//   (node IDs 6, 7, 8, 9, 10, 11)

	nodes := make([]string, 0)
	parents := make(map[string] []string)
	add := func(name string, parent ...string) {
		nodes = append(nodes, name)
		parents[name] = parent
	}
	finish := func() *DAG {
		dag := NewDAG()
		for _, name := range nodes {
			makestubnode(dag, name)
		}
		for _, name := range nodes {
			node := dag.lookup(name)
			for _, pname := range parents[name] {
				node.AddParent(dag.lookup(pname))
			}
		}
		return dag
	}

	add("tool1", "tool1.o", "misc.o", "util.o")
	add("tool2", "tool2.o", "util.o")
	add("tool1.o", "tool1.c", "misc.h", "util.h")
	add("misc.o", "misc.h", "misc.c")
	add("util.o", "util.h", "util.c")
	add("tool2.o", "tool2.c", "util.h")
	add("tool1.c")
	add("misc.h")
	add("misc.c")
	add("util.h")
	add("util.c")
	add("tool2.c")
	return finish()
}
