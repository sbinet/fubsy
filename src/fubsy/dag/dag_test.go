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

func makestubnode(dag *DAG, name string) *stubnode {
	_, node := dag.addNode(&stubnode{nodebase: makenodebase(name)})
	return node.(*stubnode)
}

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.lookup("foo")
	assert.Nil(t, outnode)

	innode := &stubnode{nodebase: makenodebase("foo")}
	_, outnode = dag.addNode(innode)
	assert.True(t, outnode == innode)
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

func Test_DAG_Rebuild_simple(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this just gives us a known set of filenames for GlobNode to search
	dag := makeSimpleGraph()
	touchSourceFiles(dag)
	// fmt.Println("after touchSourceFiles: pwd && ls -lR")
	// cmd := exec.Command("/bin/sh", "-c", "pwd && ls -lR")
	// output, err := cmd.CombinedOutput()
	// _ = err
	// fmt.Print(string(output))

	// dag.Rebuild() just copies the DAG, because it consists
	// entirely of FileNodes -- nothing to expand here
	relevant := bit.New()
	relevant.AddRange(0, len(dag.nodes))
	rdag, err := dag.Rebuild(relevant)

	assert.Nil(t, err)
	assert.False(t, &dag.nodes == &rdag.nodes)
	assert.True(t, reflect.DeepEqual(dag.nodes, rdag.nodes))
}

func Test_DAG_Rebuild_globs(t *testing.T) {
	// same setup as Test_DAG_Rebuild_simple()
	cleanup := testutils.Chtemp()
	defer cleanup()

	dag := makeSimpleGraph()
	touchSourceFiles(dag)

	dag = NewDAG()
	node0 := MakeGlobNode(dag, types.NewFileFinder([]string {"**/util.[ch]"}))
	node1 := MakeGlobNode(dag, types.NewFileFinder([]string {"*.h"}))
	node2 := MakeFileNode(dag, "util.o")
	_ = node1
	dag.addParent(node2, node0)
	assert.Equal(t, 3, dag.length())

	//fmt.Println("dag before rebuild:")
	//dag.Dump(os.Stdout)

	// relevant = {0} so we only expand the first GlobNode, and the
	// new DAG contains only nodes derived from that expansion
	relevant := bit.New(0)
	rdag, err := dag.Rebuild(relevant)

	//fmt.Println("rebuild #1:")
	//rdag.Dump(os.Stdout)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)

	buf := new(bytes.Buffer)
	dag.Dump(buf)				// no panic

	// all nodes are relevant, so the second GlobNode will be expanded
	relevant.AddRange(0, len(dag.nodes))
	rdag, err = dag.Rebuild(relevant)
	assert.Nil(t, err)

	//fmt.Println("rebuild #2:")
	//dag.Dump(os.Stdout)

	assert.Equal(t, 4, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)
	assert.Equal(t, "misc.h", rdag.nodes[2].(*FileNode).name)
	assert.Equal(t, "util.o", rdag.nodes[3].(*FileNode).name)

	// node2's parents correctly adjusted
	parents := rdag.parentNodes(node2)
	assert.Equal(t, 2, len(parents))
	assert.Equal(t, "util.c", parents[0].Name())
	assert.Equal(t, "util.h", parents[1].Name())
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
