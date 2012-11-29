package dag

import (
	"testing"
	"reflect"
	"bytes"
	//"os/exec"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"

	"fubsy/types"
	"fubsy/testutils"
)

type stubnode struct {
	nodebase
}

func (self *stubnode) Equal(other_ Node) bool {
	other, ok := other_.(*stubnode)
	return ok && self.name == other.name
}

func (self *stubnode) Changed() (bool, error) {
	return true, nil
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

func Test_DAG_replaceNode(t *testing.T) {
	dag := NewDAG()
	node0 := makestubnode(dag, "foo")
	node1 := makestubnode(dag, "bar")
	node2 := makestubnode(dag, "qux")

	dag.replaceNode(node1, []Node {})
	assert.Nil(t, dag.nodes[1])
	assert.Nil(t, dag.lookup("bar"))

	// XXX not testing parent replacement
	// (to be fair, it is tested by Test_DAG_Expand())
	assert.Equal(t, node2, dag.lookup("qux"))
	dag.replaceNode(node2, []Node {})
	assert.Nil(t, dag.nodes[2])
	assert.Nil(t, dag.lookup("qux"))

	assert.Equal(t, node0, dag.lookup("foo"))
}

func Test_DAG_FindFinalTargets(t *testing.T) {
	dag := makeSimpleGraph()
	targets := (*bit.Set)(dag.FindFinalTargets())
	assert.Equal(t, "{0, 1}", targets.String())
}

func Test_DAG_FindOriginalSources(t *testing.T) {
	dag := makeSimpleGraph()
	bstate := dag.NewBuildState()
	bstate.goal = bit.New(0, 1)		// all final targets: tool1, tool2
	bstate.FindOriginalSources()
	assert.Equal(t, "{6..11}", bstate.sources.String())
	assert.Equal(t, "{0..11}", bstate.relevant.String())

	// relevant children of misc.h = {tool1.o, misc.o}
	assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o, tool2.o}
	assert.Equal(t, "{2, 4, 5}", bstate.children[9].String())

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()
	assert.Equal(t, "{6..10}", bstate.sources.String())
	assert.Equal(t, "{0, 2..4, 6..10}", bstate.relevant.String())

	// relevant children of misc.h = {tool1.o, misc.o}
	assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o}
	assert.Equal(t, "{2, 4}", bstate.children[9].String())

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()
	assert.Equal(t, "{9..11}", bstate.sources.String())
	assert.Equal(t, "{1, 4, 5, 9..11}", bstate.relevant.String())

	// misc.h is not a relevant node, so not in children map
	assert.Nil(t, bstate.children[7])
	// relevant children of util.h = {util.o, tool2.o}
	assert.Equal(t, "{4, 5}", bstate.children[9].String())
}

func Test_DAG_FindOriginalSources_cycle(t *testing.T) {
	dag := makeSimpleGraph()
	dag.lookup("misc.h").AddParent(dag.lookup("tool1"))
	bstate := dag.NewBuildState()

	// goal = {tool2} ==> no cycle, since we don't visit those nodes
	// (this simply tests that FindOriginalSources() doesn't panic)
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()

	// goal = {tool1} ==> cycle!
	// (disabled because FindOriginalSources() currently panics on cycle)
	return
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()
}

func Test_DAG_Expand(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this just gives us a known set of filenames for GlobNode to search
	dag1 := makeSimpleGraph()
	touchSourceFiles(dag1)
	// fmt.Println("after touchSourceFiles: pwd && ls -lR")
	// cmd := exec.Command("/bin/sh", "-c", "pwd && ls -lR")
	// output, err := cmd.CombinedOutput()
	// _ = err
	// fmt.Print(string(output))

	// dag1.Expand() is a no-op, because it consists entirely of
	// FileNodes -- nothing to expand here
	relevant := bit.New()
	relevant.AddRange(0, len(dag1.nodes))
	orig := make([]Node, len(dag1.nodes))
	copy(orig, dag1.nodes)
	dag1.Expand(relevant)
	assert.True(t, reflect.DeepEqual(orig, dag1.nodes))

	dag2 := NewDAG()
	node0 := MakeGlobNode(dag2, types.NewFileFinder([]string {"**/util.[ch]"}))
	node1 := MakeGlobNode(dag2, types.NewFileFinder([]string {"*.h"}))
	node2 := MakeFileNode(dag2, "util.o")
	node2.AddParent(node0)
	assert.Equal(t, 3, dag2.length())

	// relevant = {0} so we only expand the first GlobNode
	relevant = bit.New(0)
	dag2.Expand(relevant)
	assert.Equal(t, 5, len(dag2.nodes))
	assert.Nil(t, dag2.nodes[0])
	assert.Equal(t, node1, dag2.nodes[1])
	assert.Equal(t, node2, dag2.nodes[2])
	assert.Equal(t, "util.c", dag2.nodes[3].(*FileNode).name)
	assert.Equal(t, "util.h", dag2.nodes[4].(*FileNode).name)
	buf := new(bytes.Buffer)

	// node2's parents correctly adjusted
	parents := node2.Parents()
	assert.Equal(t, 2, len(parents))
	assert.Equal(t, "util.c", parents[0].Name())
	assert.Equal(t, "util.h", parents[1].Name())

	dag2.Dump(buf)				// no panic on nil nodes

	// all nodes are relevant, so the second GlobNode will be expanded
	relevant.AddRange(0, len(dag2.nodes))
	dag2.Expand(relevant)

	assert.Equal(t, 6, len(dag2.nodes))
	assert.Nil(t, dag2.nodes[0])
	assert.Nil(t, dag2.nodes[1])
	assert.Equal(t, "util.c", dag2.nodes[3].(*FileNode).name)
	assert.Equal(t, "util.h", dag2.nodes[4].(*FileNode).name)
	assert.Equal(t, "misc.h", dag2.nodes[5].(*FileNode).name)
}

func Test_DAG_FindStaleTargets(t *testing.T) {
	// this test depends on FindOriginalSources() working
	// goal = {tool1}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	// initial rebuild = {tool1.o, misc.o, util.o}
	dag := makeSimpleGraph()
	bstate := dag.NewBuildState()
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()

	errors := bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{2..4}", bstate.rebuild.String())

	// goal = {tool2}
	// sources = {tool2.c, util.h, util.c}
	// initial rebuild = {tool2.o, util.o}
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()

	errors = bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{4, 5}", bstate.rebuild.String())

	// goal = {tool1, tool2}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c, tool2.c}
	// initial rebuild = {tool1.o, misc.o, util.o, tool2.o}
	bstate.goal = bit.New(0, 1)
	bstate.FindOriginalSources()
	errors = bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{2..5}", bstate.rebuild.String())
}

func makeSimpleGraph() *DAG {
	// dependency graph for a simple C project
	//    0: tool1:		{tool1.o, misc.o, util.o}
	//    1: tool2:		{tool2.o, util.o}
	//    2: tool1.o:	{tool1.c, misc.h, util.h}
	//    3: misc.o:	{misc.h, misc.c}
	//    4: util.o:	{util.h, util.c}
	//    5: tool2.o:	{tool2.c, util.h}
	//    6: tool1.c:	{}
	//    7: misc.h:	{}
	//    8: misc.c:	{}
	//    9: util.h:	{}
	//   10: util.c:	{}
	//   11: tool2.c:	{}
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

func touchSourceFiles(dag *DAG) {
	filenames := []string {}
	for _, node := range dag.nodes {
		if (*bit.Set)(node.ParentSet()).IsEmpty() {
			filenames = append(filenames, node.Name())
		}
	}
	testutils.TouchFiles(filenames...)
}
