// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"bytes"
	"errors"
	"reflect"
	"strconv"
	"testing"

	"code.google.com/p/go-bit/bit"
	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_DAG_add_lookup(t *testing.T) {
	dag := NewDAG()
	outnode := dag.lookup("foo")
	assert.Nil(t, outnode)

	innode := NewStubNode("foo")
	_, outnode = dag.addNode(innode)
	assert.True(t, outnode == innode)
	assert.True(t, innode == dag.nodes[0].(*StubNode))

	outnode = dag.lookup("foo")
	assert.True(t, outnode.(*StubNode) == innode)

	assert.Nil(t, dag.lookup("bar"))
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

	tdag := newtestdag()
	tdag.add("a", "c", "f", "h")
	tdag.add("b", "d", "f", "g", "h")
	tdag.add("c", "b", "e")
	tdag.add("d", "g")
	tdag.add("e")
	tdag.add("f", "g")
	tdag.add("g")
	tdag.add("h")
	dag := tdag.finish()

	actual := []string{}
	visit := func(id int) error {
		actual = append(actual, dag.nodes[id].Name())
		return nil
	}

	assertDFS := func(start *bit.Set, expect []string) {
		actual = actual[0:0]
		dag.DFS(start, visit)
		assert.Equal(t, expect, actual)
	}

	start := bit.New()
	start.AddRange(0, 7)
	expect := []string{"g", "d", "f", "h", "b", "e", "c", "a"}
	assertDFS(start, expect)

	// start nodes: {c, f}
	start = bit.New(2, 5)
	expect = []string{"g", "d", "f", "h", "b", "e", "c"}
	assertDFS(start, expect)

	// start nodes: {d, f}
	start = bit.New(3, 5)
	expect = []string{"g", "d", "f"}
	assertDFS(start, expect)
}

func Test_DAG_DFS_cycle(t *testing.T) {
	var tdag *testdag
	var dag *DAG
	var err error

	visit := func(id int) error { return nil }

	tdag = newtestdag()
	tdag.add("0", "1")
	tdag.add("1", "0")
	dag = tdag.finish()
	err = dag.DFS(NodeSet(bit.New(0, 1)), visit)
	assert.Equal(t, "found 1 dependency cycles:\n  0 -> 1 -> 0", err.Error())

	// degenerate case
	tdag = newtestdag()
	tdag.add("0", "0")
	dag = tdag.finish()
	err = dag.DFS(NodeSet(bit.New(0)), visit)
	assert.Equal(t, "found 1 dependency cycles:\n  0 -> 0", err.Error())

	// weird case: two disconnected isomorphic graphs stuck in the
	// same data structure; each has two discernible cycles:
	//   0 -> 2 -> 4 -> 6 -> 2
	//   0 -> 2 -> 4 -> 8 -> 0
	//   1 -> 3 -> 5 -> 7 -> 3
	//   1 -> 3 -> 5 -> 9 -> 1
	tdag = newtestdag()
	tdag.add("0", "2", "4")
	tdag.add("1", "3", "5")
	tdag.add("2", "4")
	tdag.add("3", "5")
	tdag.add("4", "6", "8")
	tdag.add("5", "7", "9")
	tdag.add("6", "2")
	tdag.add("7", "3")
	tdag.add("8", "0")
	tdag.add("9", "1")
	dag = tdag.finish()
	start := bit.New()
	start.AddRange(0, 9)
	err = dag.DFS(NodeSet(start), visit)
	cycerr := err.(CycleError)
	assert.Equal(t, 4, len(cycerr.cycles))
	assert.Equal(t, []int{0, 2, 4, 6, 2}, cycerr.cycles[0])
	assert.Equal(t, []int{0, 2, 4, 8, 0}, cycerr.cycles[1])
	assert.Equal(t, []int{1, 3, 5, 7, 3}, cycerr.cycles[2])
	assert.Equal(t, []int{1, 3, 5, 9, 1}, cycerr.cycles[3])
}

func Test_DAG_DFS_error(t *testing.T) {
	tdag := newtestdag()
	tdag.add("0", "2", "4")
	tdag.add("1", "3", "5")
	tdag.add("2", "4")
	tdag.add("3", "5")
	tdag.add("4", "3", "2") // dependency cycle goes unreported
	tdag.add("5")
	dag := tdag.finish()

	visited := make([]int, 0)
	visit := func(id int) error {
		visited = append(visited, id)
		if id == 4 {
			return errors.New("bite me")
		}
		return nil
	}

	err := dag.DFS(bit.New(0), visit)
	assert.Equal(t, "bite me", err.Error())
	assert.Equal(t, []int{5, 3, 4}, visited)
}

func Test_DAG_FindRelevantNodes(t *testing.T) {
	dag := makeSimpleGraph()
	goal := bit.New(0, 1) // all final targets: tool1, tool2
	relevant := dag.FindRelevantNodes(NodeSet(goal))
	assert.Equal(t, "{0,1,2,3,4,5,6,7,8,9,10,11}", setToString(relevant))

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	goal = bit.New(0)
	relevant = dag.FindRelevantNodes(NodeSet(goal))
	assert.Equal(t, "{0,2,3,4,6,7,8,9,10}", setToString(relevant))

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	goal = bit.New(1)
	relevant = dag.FindRelevantNodes(NodeSet(goal))
	assert.Equal(t, "{1,4,5,9,10,11}", setToString(relevant))
}

func Test_DAG_MarkSources(t *testing.T) {
	dag := makeSimpleGraph()

	// initial sanity check
	assert.Equal(t, UNKNOWN, dag.lookup("tool1.c").State())
	assert.Equal(t, UNKNOWN, dag.lookup("tool1.o").State())
	assert.Equal(t, UNKNOWN, dag.lookup("tool1").State())

	dag.MarkSources()
	assert.Equal(t, SOURCE, dag.lookup("tool1.c").State())
	assert.Equal(t, UNKNOWN, dag.lookup("tool1.o").State())
	assert.Equal(t, UNKNOWN, dag.lookup("tool1").State())
}

func Test_DAG_Rebuild_simple(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this just gives us a known set of filenames for FinderNode to search
	dag := makeSimpleGraph()

	// dag.Rebuild() just copies the DAG, because it consists
	// entirely of FileNodes -- nothing to expand here
	relevant := bit.New()
	relevant.AddRange(0, len(dag.nodes))
	ns := types.NewValueMap()
	rdag, err := dag.Rebuild(relevant, ns)

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
	node0 := MakeFinderNode(dag, []string{"**/util.[ch]"})
	node1 := MakeFinderNode(dag, []string{"*.h"})
	node2 := MakeFileNode(dag, "util.o")
	_ = node1
	dag.addParent(node2, node0)
	assert.Equal(t, 3, dag.length())

	// relevant = {0} so we only expand the first FinderNode, and the
	// new DAG contains only nodes derived from that expansion
	relevant := bit.New(0)
	ns := types.NewValueMap()
	rdag, err := dag.Rebuild(relevant, ns)

	assert.Nil(t, err)
	assert.Equal(t, 2, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)

	buf := new(bytes.Buffer)
	dag.Dump(buf, "") // no panic

	// all nodes are relevant, so the second FinderNode will be expanded
	relevant.AddRange(0, len(dag.nodes))
	ns = types.NewValueMap()
	rdag, err = dag.Rebuild(relevant, ns)
	assert.Nil(t, err)

	assert.Equal(t, 4, len(rdag.nodes))
	assert.Equal(t, "util.c", rdag.nodes[0].(*FileNode).name)
	assert.Equal(t, "util.h", rdag.nodes[1].(*FileNode).name)
	assert.Equal(t, "misc.h", rdag.nodes[2].(*FileNode).name)
	assert.Equal(t, "util.o", rdag.nodes[3].(*FileNode).name)

	// parents of node2 (util.o) were correctly adjusted
	parents := rdag.parentNodes(3)
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

	tdag := newtestdag()
	tdag.add("tool1", "tool1.o", "misc.o", "util.o")
	tdag.add("tool2", "tool2.o", "util.o")
	tdag.add("tool1.o", "tool1.c", "misc.h", "util.h")
	tdag.add("misc.o", "misc.h", "misc.c")
	tdag.add("util.o", "util.h", "util.c")
	tdag.add("tool2.o", "tool2.c", "util.h")
	tdag.add("tool1.c")
	tdag.add("misc.h")
	tdag.add("misc.c")
	tdag.add("util.h")
	tdag.add("util.c")
	tdag.add("tool2.c")
	return tdag.finish()
}

// string-based DAG that's easy to construct, and then gets converted
// to the real thing
type testdag struct {
	nodes   []string
	parents map[string][]string
}

func newtestdag() *testdag {
	return &testdag{parents: make(map[string][]string)}
}

func (self *testdag) add(name string, parent ...string) {
	self.nodes = append(self.nodes, name)
	self.parents[name] = parent
}

func (self *testdag) finish() *DAG {
	dag := NewDAG()
	for _, name := range self.nodes {
		MakeStubNode(dag, name)
	}
	for _, name := range self.nodes {
		node := dag.lookup(name)
		for _, pname := range self.parents[name] {
			dag.addParent(node, dag.lookup(pname))
		}
	}
	return dag
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

func newNodeSet(dag *DAG, names ...string) NodeSet {
	set := bit.New()
	for _, name := range names {
		if id, ok := dag.index[name]; ok {
			set.Add(id)
		} else {
			panic("no such node in DAG: " + name)
		}
	}
	return NodeSet(set)
}

// test-friendly way to formatting a NodeSet as a string
func setToString(set_ NodeSet) string {
	set := (*bit.Set)(set_)
	result := make([]byte, 1, set.Size()*3)
	result[0] = '{'
	set.Do(func(n int) {
		result = strconv.AppendInt(result, int64(n), 10)
		result = append(result, ',')
	})
	result[len(result)-1] = '}'
	return string(result)
}
