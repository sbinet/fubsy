package dag

import (
	"testing"
	"reflect"
	"bytes"
	//"fmt"
	//"os"
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

func (self *stubnode) addParent(parent Node) {
	self.dag.addParent(self, parent)
}

func makestubnode(dag *DAG, name string) *stubnode {
	node := dag.lookup(name)
	if node == nil {
		node := &stubnode{
			nodebase: makenodebase(dag, name),
		}
		dag.addNode(node)
		return node
	}
	return node.(*stubnode)
}

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.lookup("foo")
	assert.Nil(t, outnode)

	innode := &stubnode{nodebase: makenodebase(dag, "foo")}
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
	dag2.addParent(node2, node0)
	assert.Equal(t, 3, dag2.length())

	// fmt.Println("dag2 before expansion:")
	// dag2.Dump(os.Stdout)

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

	// fmt.Println("\ndag2 after expansion #1:")
	// dag2.Dump(os.Stdout)

	// node2's parents correctly adjusted
	parents := dag2.parentNodes(node2)
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
				dag.addParent(node, dag.lookup(pname))
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
	for id, node := range dag.nodes {
		if dag.parents[id].IsEmpty() {
			filenames = append(filenames, node.Name())
		}
	}
	testutils.TouchFiles(filenames...)
}
