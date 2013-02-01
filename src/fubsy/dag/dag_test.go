// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"bytes"
	"errors"
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

func Test_DAG_ExpandNodes(t *testing.T) {
	ns := types.NewValueMap()
	ns.Assign("sdir", types.FuString("src/tool1"))
	ns.Assign("ext", types.FuString("cpp"))

	tdag := NewTestDAG()
	tdag.Add("bin/tool1", "$sdir/main.$ext", "$sdir/util.$ext", "$sdir/foo.h")
	tdag.Add("$sdir/foo.h", "$sdir/foo.h.in")
	tdag.Add("$sdir/main.$ext")
	tdag.Add("$sdir/util.$ext", "$sdir/foo.h")
	tdag.Add("$sdir/foo.h.in")
	dag := tdag.Finish()

	expect := []string{
		"bin/tool1",
		"src/tool1/foo.h",
		"src/tool1/main.cpp",
		"src/tool1/util.cpp",
		"src/tool1/foo.h.in",
	}
	errs := dag.ExpandNodes(ns)
	assert.Equal(t, 0, len(errs))
	for i, node := range dag.nodes {
		assert.Equal(t, expect[i], node.Name())
	}
	assert.Equal(t, len(expect), len(dag.nodes))

	tdag = NewTestDAG()
	tdag.Add("foo/$bogus/blah", "bam")
	tdag.Add("bam", "$flop/bop")
	tdag.Add("$flop/bop")
	dag = tdag.Finish()
	errs = dag.ExpandNodes(ns)
	if len(errs) == 2 {
		assert.Equal(t, "undefined variable 'bogus' in string", errs[0].Error())
		assert.Equal(t, "undefined variable 'flop' in string", errs[1].Error())
	} else {
		t.Errorf("expected %d errors, but got %d:\n%v", 2, len(errs), errs)
	}
}

func Test_DAG_MatchTargets(t *testing.T) {
	tdag := NewTestDAG()
	tdag.Add("foo/bar1", "s1", "s2", "s3")
	tdag.Add("foo/bar2", "s2")
	tdag.Add("bar/foo1", "s1", "s2")
	tdag.Add("bar/foox", "s1", "s3")
	tdag.Add("final", "foo/bar1", "foo/bar2", "bar/foo1", "bar/foox")
	tdag.Add("s1")
	tdag.Add("s2")
	tdag.Add("s3")
	dag := tdag.Finish()

	tests := []struct {
		name   string
		expect string
		errmsg string
	}{
		{"final", "{4}", ""},
		{"fo", "{}", "no targets found matching 'fo'"},
		{"f", "{}", "no targets found matching 'f'"},
		{"foo", "{0,1}", ""},
		{"foo/", "{0,1}", ""},
		{"foo/bar", "{}", "no targets found matching 'foo/bar'"},
		{"foo/bar1", "{0}", ""},
		{"s1", "{}", "not a target: 's1'"},
		{"s", "{}", "no targets found matching 's'"},
	}
	for _, test := range tests {
		match, errs := dag.MatchTargets([]string{test.name})
		if test.errmsg == "" {
			assert.Equal(t, 0, len(errs),
				"expected no errors, but got %v", errs)
			actual := match.String()
			assert.Equal(t, test.expect, actual)
		} else {
			if len(errs) == 1 {
				assert.Equal(t, test.errmsg, errs[0].Error())
			} else {
				t.Errorf(
					"target prefix %s: expected exactly one error, but got %v",
					test.name, errs)
			}
		}
	}

	// Now make sure that passing multiple patterns can return
	// multiple errors.
	match, errs := dag.MatchTargets(
		[]string{"foo/", "bar/foo", "s2", "final", "s"})
	assert.Equal(t, "{0,1,4}", match.String())
	if len(errs) == 3 {
		assert.Equal(t, "no targets found matching 'bar/foo'", errs[0].Error())
		assert.Equal(t, "not a target: 's2'", errs[1].Error())
		assert.Equal(t, "no targets found matching 's'", errs[2].Error())
	} else {
		t.Errorf("expected 3 errors, but got %d:\n%v", len(errs), errs)
	}
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

func Test_DAG_DFS_error_hides_cycle(t *testing.T) {
	tdag := NewTestDAG()
	tdag.Add("0", "2", "4")
	tdag.Add("1", "3", "5")
	tdag.Add("2", "4")
	tdag.Add("3", "5")
	tdag.Add("4", "3", "2") // dependency cycle goes unreported
	tdag.Add("5")
	dag := tdag.Finish()

	visited := []int{}
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

func Test_DAG_DFS_error_stop_early(t *testing.T) {
	// similar, but this time the error is in a top-level node (no
	// children), and it cuts the DFS short before we get to the
	// second top-level node
	tdag := NewTestDAG()
	tdag.Add("0", "2", "3")
	tdag.Add("1", "3")
	tdag.Add("2", "3")
	tdag.Add("3")
	dag := tdag.Finish()

	visited := []int{}
	visit := func(node Node) error {
		id := node.id()
		visited = append(visited, id)
		if id == 0 {
			return errors.New("fail")
		}
		return nil
	}

	err := dag.DFS(dag.MakeNodeSet("0", "1"), visit)
	assert.Equal(t, "fail", err.Error())
	assert.Equal(t, []int{3, 2, 0}, visited)
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
	expect := `0000: StubNode 0 (state UNKNOWN)
  parents:
    0001: 1
    0002: 2
    0003: 3
0001: StubNode 1 (state UNKNOWN)
  parents:
    0003: 3
0002: StubNode 2 (state UNKNOWN)
  parents:
    0003: 3
0003: StubNode 3 (state UNKNOWN)
`
	actual := string(buf.Bytes())
	if expect != actual {
		t.Errorf("expected:\n%s\nbut got:\n%s", expect, actual)
	}
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
