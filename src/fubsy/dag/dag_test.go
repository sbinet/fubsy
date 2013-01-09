// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"code.google.com/p/go-bit/bit"
	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.Lookup("foo")
	assert.Nil(t, outnode)

	innode := NewStubNode("foo")
	_, outnode = dag.addNode(innode)
	assert.True(t, outnode == innode)
	assert.True(t, innode == dag.nodes[0].(*StubNode))

	outnode = dag.Lookup("foo")
	assert.True(t, outnode.(*StubNode) == innode)

	assert.Nil(t, dag.Lookup("bar"))
}

func Test_DAG_FindFinalTargets(t *testing.T) {
	dag := makeSimpleGraph()
	targets := (*bit.Set)(dag.FindFinalTargets())
	assert.Equal(t, "{0, 1}", targets.String())
}

func Test_DAG_DFS(t *testing.T) {
	// 0: a -> {c, f, h}
	// 1: b -> {d, f, g, h}
	// 2: c -> {b, e}
	// 3: d -> {g}
	// 4: e -> {}
	// 5: f -> {g}
	// 6: g -> {}
	// 7: h -> {}
	// original sources: {e, g, h} = {4, 6, 7}
	// final targets:    {a} = {0}

	tdag := NewTestDAG()
	tdag.Add("a", "c", "f", "h")
	tdag.Add("b", "d", "f", "g", "h")
	tdag.Add("c", "b", "e")
	tdag.Add("d", "g")
	tdag.Add("e")
	tdag.Add("f", "g")
	tdag.Add("g")
	tdag.Add("h")
	dag := tdag.Finish()

	actual := []string{}
	visit := func(node Node) error {
		actual = append(actual, node.Name())
		return nil
	}

	assertDFS := func(start *NodeSet, expect []string) {
		actual = actual[0:0]
		dag.DFS(start, visit)
		assert.Equal(t, expect, actual)
	}

	start := dag.MakeNodeSet("a", "h")
	expect := []string{"g", "d", "f", "h", "b", "e", "c", "a"}
	assertDFS(start, expect)

	start = dag.MakeNodeSet("c", "f")
	expect = []string{"g", "d", "f", "h", "b", "e", "c"}
	assertDFS(start, expect)

	start = dag.MakeNodeSet("d", "f")
	expect = []string{"g", "d", "f"}
	assertDFS(start, expect)
}

func Test_DAG_DFS_cycle(t *testing.T) {
	var tdag *TestDAG
	var dag *DAG
	var err error

	visit := func(node Node) error { return nil }

	tdag = NewTestDAG()
	tdag.Add("0", "1")
	tdag.Add("1", "0")
	dag = tdag.Finish()
	start := dag.MakeNodeSet("0", "1")
	err = dag.DFS(start, visit)
	assert.Equal(t, "found 1 dependency cycles:\n  0 -> 1 -> 0", err.Error())

	// degenerate case
	tdag = NewTestDAG()
	tdag.Add("0", "0")
	dag = tdag.Finish()
	start = dag.MakeNodeSet("0")
	err = dag.DFS(start, visit)
	assert.Equal(t, "found 1 dependency cycles:\n  0 -> 0", err.Error())

	// weird case: two disconnected isomorphic graphs stuck in the
	// same data structure; each has two discernible cycles:
	//   0 -> 2 -> 4 -> 6 -> 2
	//   0 -> 2 -> 4 -> 8 -> 0
	//   1 -> 3 -> 5 -> 7 -> 3
	//   1 -> 3 -> 5 -> 9 -> 1
	tdag = NewTestDAG()
	tdag.Add("0", "2", "4")
	tdag.Add("1", "3", "5")
	tdag.Add("2", "4")
	tdag.Add("3", "5")
	tdag.Add("4", "6", "8")
	tdag.Add("5", "7", "9")
	tdag.Add("6", "2")
	tdag.Add("7", "3")
	tdag.Add("8", "0")
	tdag.Add("9", "1")
	dag = tdag.Finish()
	start = dag.MakeNodeSet()
	(*bit.Set)(start).AddRange(0, 9)
	err = dag.DFS(start, visit)
	cycerr := err.(CycleError)
	assert.Equal(t, 4, len(cycerr.cycles))
	assert.Equal(t, []int{0, 2, 4, 6, 2}, cycerr.cycles[0])
	assert.Equal(t, []int{0, 2, 4, 8, 0}, cycerr.cycles[1])
	assert.Equal(t, []int{1, 3, 5, 7, 3}, cycerr.cycles[2])
	assert.Equal(t, []int{1, 3, 5, 9, 1}, cycerr.cycles[3])
}

func Test_DAG_DFS_error(t *testing.T) {
	tdag := NewTestDAG()
	tdag.Add("0", "2", "4")
	tdag.Add("1", "3", "5")
	tdag.Add("2", "4")
	tdag.Add("3", "5")
	tdag.Add("4", "3", "2") // dependency cycle goes unreported
	tdag.Add("5")
	dag := tdag.Finish()

	visited := make([]int, 0)
	visit := func(node Node) error {
		id := node.id()
		visited = append(visited, id)
		if id == 4 {
			return errors.New("bite me")
		}
		return nil
	}

	err := dag.DFS(dag.MakeNodeSet("0"), visit)
	assert.Equal(t, "bite me", err.Error())
	assert.Equal(t, []int{5, 3, 4}, visited)
}

func Test_DAG_AddManyParents(t *testing.T) {
	dag := NewDAG()
	node0 := MakeStubNode(dag, "0")
	node1 := MakeStubNode(dag, "1")
	node2 := MakeStubNode(dag, "2")
	node3 := MakeStubNode(dag, "3")

	dag.AddManyParents([]Node{node0}, []Node{node1, node2, node3})
	dag.AddManyParents([]Node{node1, node2}, []Node{node3})
	buf := &bytes.Buffer{}
	dag.Dump(buf, "")
	expect := `0000: 0 (StubNode, state UNKNOWN)
  parents:
    0001: 1
    0002: 2
    0003: 3
0001: 1 (StubNode, state UNKNOWN)
  parents:
    0003: 3
0002: 2 (StubNode, state UNKNOWN)
  parents:
    0003: 3
0003: 3 (StubNode, state UNKNOWN)
`
	actual := string(buf.Bytes())
	assert.Equal(t, expect, actual,
		"expected:\n%s\nbut got:\n%s", expect, actual)
}

func Test_DAG_FindRelevantNodes(t *testing.T) {
	dag := makeSimpleGraph()
	goal := dag.MakeNodeSet("tool1", "tool2")
	relevant := dag.FindRelevantNodes(goal)
	assert.Equal(t, "{0,1,2,3,4,5,6,7,8,9,10,11}", relevant.String())

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	goal = dag.MakeNodeSet("tool1")
	relevant = dag.FindRelevantNodes(goal)
	assert.Equal(t, "{0,2,3,4,6,7,8,9,10}", relevant.String())

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	goal = dag.MakeNodeSet("tool2")
	relevant = dag.FindRelevantNodes(goal)
	assert.Equal(t, "{1,4,5,9,10,11}", relevant.String())
}

func Test_DAG_MarkSources(t *testing.T) {
	dag := makeSimpleGraph()

	// initial sanity check
	assert.Equal(t, UNKNOWN, dag.Lookup("tool1.c").State())
	assert.Equal(t, UNKNOWN, dag.Lookup("tool1.o").State())
	assert.Equal(t, UNKNOWN, dag.Lookup("tool1").State())

	dag.MarkSources()
	assert.Equal(t, SOURCE, dag.Lookup("tool1.c").State())
	assert.Equal(t, UNKNOWN, dag.Lookup("tool1.o").State())
	assert.Equal(t, UNKNOWN, dag.Lookup("tool1").State())
}

func Test_DAG_Rebuild_simple(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this just gives us a known set of filenames for FinderNode to search
	dag := makeSimpleGraph()
	orig := dag.copy()

	// dag.Rebuild() just copies the DAG, because it consists
	// entirely of FileNodes -- nothing to expand here
	relevant := dag.MakeNodeSet()
	(*bit.Set)(relevant).AddRange(0, len(dag.nodes))
	ns := types.NewValueMap()
	rdag, err := dag.Rebuild(relevant, ns)

	assert.Nil(t, err)
	assert.False(t, &dag.nodes == &rdag.nodes)
	assert.True(t, reflect.DeepEqual(orig.nodes, rdag.nodes))
}

func Test_DAG_Rebuild_globs(t *testing.T) {
	// same setup as Test_DAG_Rebuild_simple()
	cleanup := testutils.Chtemp()
	defer cleanup()

	dag := makeSimpleGraph()
	touchSourceFiles(dag)

	dag = NewDAG()
	var node0, node1, node2 Node
	node0 = MakeFinderNode(dag, "**/util.[ch]")
	node1 = MakeFinderNode(dag, "*.h")
	node2 = MakeFileNode(dag, "util.o")
	_ = node1
	dag.AddParent(node2, node0)
	assert.Equal(t, 3, dag.length())
	dag.verify()

	savedag := dag.copy()

	// relevant = {0} so we only expand the first FinderNode, and the
	// new DAG contains only nodes derived from that expansion
	relevant := dag.MakeNodeSet(node0.Name())
	ns := types.NewValueMap()
	rdag, err := dag.Rebuild(relevant, ns)
	savedag.verify()

	assert.Nil(t, err)
	assert.Equal(t, 2, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)

	buf := new(bytes.Buffer)
	dag.Dump(buf, "") // no panic

	// second rebuild where all nodes are relevant, so the second
	// FinderNode will be expanded
	dag = savedag
	dag.verify()
	node2 = dag.nodes[2] // need the pre-Rebuild() copy
	(*bit.Set)(relevant).AddRange(0, len(dag.nodes))
	ns = types.NewValueMap()
	rdag, err = dag.Rebuild(relevant, ns)
	assert.Nil(t, err)

	assert.Equal(t, 4, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)
	assert.Equal(t, "misc.h", rdag.nodes[2].(*FileNode).name)
	assert.Equal(t, "util.o", rdag.nodes[3].(*FileNode).name)

	// parents of node2 (util.o) were correctly adjusted
	parents := rdag.ParentNodes(node2)
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

	tdag := NewTestDAG()
	tdag.Add("tool1", "tool1.o", "misc.o", "util.o")
	tdag.Add("tool2", "tool2.o", "util.o")
	tdag.Add("tool1.o", "tool1.c", "misc.h", "util.h")
	tdag.Add("misc.o", "misc.h", "misc.c")
	tdag.Add("util.o", "util.h", "util.c")
	tdag.Add("tool2.o", "tool2.c", "util.h")
	tdag.Add("tool1.c")
	tdag.Add("misc.h")
	tdag.Add("misc.c")
	tdag.Add("util.h")
	tdag.Add("util.c")
	tdag.Add("tool2.c")
	return tdag.Finish()
}

func touchSourceFiles(dag *DAG) {
	filenames := []string{}
	for id, node := range dag.nodes {
		if dag.parents[id].IsEmpty() {
			filenames = append(filenames, node.Name())
		}
	}
	testutils.TouchFiles(filenames...)
}
