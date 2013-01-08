// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"testing"

	"code.google.com/p/go-bit/bit"
	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
)

// full build (all targets), all actions succeed
func Test_BuildState_BuildTargets_full_success(t *testing.T) {
	graph, executed := setupBuild()

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.BUILT},
		{"util.o", dag.BUILT},
		{"tool1", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	bstate := NewBuildState(graph, BuildOptions{})
	goal := dag.NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)

	assert.Equal(t, dag.SOURCE, graph.Lookup("tool2.c").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("misc.h").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("util.c").State())
}

// full build (all targets), one action fails
func Test_BuildState_BuildTargets_full_failure(t *testing.T) {
	graph, executed := setupBuild()

	// fail to build misc.{c,h} -> misc.o: that will block building
	// tool1, but not tool2 (since keepGoing() always returns true
	// (for now))
	rule := graph.Lookup("misc.o").BuildRule().(*dag.StubRule)
	rule.SetFail(true)

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.FAILED},
		{"util.o", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	opts := BuildOptions{}
	bstate := NewBuildState(graph, opts)
	goal := dag.NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, graph, expect, *executed)

	// we don't even look at tool1, since an earlier node failed and
	// the build terminates on first failure
	assert.Equal(t, dag.UNKNOWN, graph.Lookup("tool1").State())
}

// full build (all targets), one action fails, --keep-going true
func Test_BuildState_BuildTargets_full_failure_keep_going(t *testing.T) {
	// this is the same as the previous test except that
	// opts.KeepGoing == true: we don't terminate the build on first
	// failure, but carry on and consider building tool1, then mark it
	// TAINTED because one of its ancestors (misc.o) failed to build

	graph, executed := setupBuild()

	rule := graph.Lookup("misc.o").BuildRule().(*dag.StubRule)
	rule.SetFail(true)

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.FAILED},
		{"util.o", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	opts := BuildOptions{KeepGoing: true}
	bstate := NewBuildState(graph, opts)
	goal := dag.NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, graph, expect, *executed)

	assert.Equal(t, dag.TAINTED, graph.Lookup("tool1").State())
}

func setupBuild() (*dag.DAG, *[]string) {
	graph := makeSimpleGraph()

	// add a stub build rule to every target node, so we track when
	// each rule's Execute() method is called
	executed := []string{}
	callback := func(name string) {
		executed = append(executed, name)
	}
	for _, node := range graph.Nodes() {
		if graph.HasParents(node) {
			rule := dag.MakeStubRule(callback, node)
			node.SetBuildRule(rule)
		}
	}

	graph.MarkSources()

	// need to return a pointer to the executed slice because
	// callback() modifies the slice
	return graph, &executed
}

func makeSimpleGraph() *dag.DAG {
	// this is the same as makeSimpleGraph() in ../dag/dag_test.go; it
	// would be nice to keep them in sync for as long as possible ...
	// but eventually they will drift out of sync, so don't kill
	// yourself trying to keep them the same
	tdag := dag.NewTestDAG()
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

func assertBuild(
	t *testing.T,
	dag *dag.DAG,
	expect []buildexpect,
	executed []string) {
	assert.Equal(t, len(expect), len(executed),
		"expected %d build attempts (%v), but got %d",
		len(expect), expect, len(executed))
	for i, expect := range expect {
		assert.Equal(t, expect.name, executed[i],
			"action %d: expected %s", i, expect.name)
		actualstate := dag.Lookup(expect.name).State()
		assert.Equal(t, expect.state, actualstate,
			"target: %s (state = %v)", expect.name, actualstate)
	}
}

type buildexpect struct {
	name  string
	state dag.NodeState
}

func Test_BuildError_Error(t *testing.T) {
	graph := dag.NewDAG()
	err := &BuildError{}
	err.attempts = 43
	err.failed = mknodelist(
		graph, "foo", "bar", "baz")
	assert.Equal(t,
		"failed to build 3 of 43 targets: foo, bar, baz", err.Error())
	err.attempts = -1
	assert.Equal(t,
		"failed to build target: foo", err.Error())

	err.attempts = 17
	err.failed = mknodelist(
		graph, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k")
	assert.Equal(t,
		"failed to build 11 of 17 targets: a, b, c, d, e, f, g, h, i, ...",
		err.Error())
}

func Test_joinNodes(t *testing.T) {
	graph := dag.NewDAG()
	nodes := mknodelist(graph, "blargh", "merp", "whoosh", "fwob", "whee")

	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 10, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 5, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, ...", joinNodes(", ", 4, nodes))
	assert.Equal(t,
		"blargh!*!merp!*!...", joinNodes("!*!", 3, nodes))
}

func mknodelist(graph *dag.DAG, names ...string) []dag.Node {
	result := make([]dag.Node, len(names))
	for i, name := range names {
		result[i] = dag.MakeStubNode(graph, name)
	}
	return result
}
