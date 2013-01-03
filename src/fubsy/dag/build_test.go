// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"testing"

	"code.google.com/p/go-bit/bit"
	"github.com/stretchrcom/testify/assert"
)

// full build (all targets), all actions succeed
func Test_BuildState_BuildTargets_full_success(t *testing.T) {
	dag, executed := setupBuild()

	expect := []buildexpect{
		{"tool1.o", BUILT},
		{"misc.o", BUILT},
		{"util.o", BUILT},
		{"tool1", BUILT},
		{"tool2.o", BUILT},
		{"tool2", BUILT},
	}

	bstate := dag.NewBuildState(BuildOptions{})
	goal := NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, dag, expect, *executed)

	assert.Equal(t, SOURCE, dag.lookup("tool2.c").State())
	assert.Equal(t, SOURCE, dag.lookup("misc.h").State())
	assert.Equal(t, SOURCE, dag.lookup("util.c").State())
}

// full build (all targets), one action fails
func Test_BuildState_BuildTargets_full_failure(t *testing.T) {
	dag, executed := setupBuild()

	// fail to build misc.{c,h} -> misc.o: that will block building
	// tool1, but not tool2 (since keepGoing() always returns true
	// (for now))
	rule := dag.lookup("misc.o").BuildRule().(*stubrule)
	rule.fail = true

	expect := []buildexpect{
		{"tool1.o", BUILT},
		{"misc.o", FAILED},
		{"util.o", BUILT},
		{"tool2.o", BUILT},
		{"tool2", BUILT},
	}

	opts := BuildOptions{}
	bstate := dag.NewBuildState(opts)
	goal := NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, dag, expect, *executed)

	// we don't even look at tool1, since an earlier node failed and
	// the build terminates on first failure
	assert.Equal(t, UNKNOWN, dag.lookup("tool1").State())
}

// full build (all targets), one action fails, --keep-going true
func Test_BuildState_BuildTargets_full_failure_keep_going(t *testing.T) {
	// this is the same as the previous test except that
	// opts.KeepGoing == true: we don't terminate the build on first
	// failure, but carry on and consider building tool1, then mark it
	// TAINTED because one of its ancestors (misc.o) failed to build

	dag, executed := setupBuild()

	rule := dag.lookup("misc.o").BuildRule().(*stubrule)
	rule.fail = true

	expect := []buildexpect{
		{"tool1.o", BUILT},
		{"misc.o", FAILED},
		{"util.o", BUILT},
		{"tool2.o", BUILT},
		{"tool2", BUILT},
	}

	opts := BuildOptions{KeepGoing: true}
	bstate := dag.NewBuildState(opts)
	goal := NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, dag, expect, *executed)

	assert.Equal(t, TAINTED, dag.lookup("tool1").State())
}

func setupBuild() (*DAG, *[]string) {
	dag := makeSimpleGraph()

	// add a stub build rule to every target node, so we track when
	// each rule's Execute() method is called
	executed := []string{}
	callback := func(name string) {
		executed = append(executed, name)
	}
	for id, node := range dag.nodes {
		if !dag.parents[id].IsEmpty() {
			rule := makestubrule(callback, node)
			node.SetBuildRule(rule)
		}
	}

	dag.MarkSources()

	// need to return a pointer to the executed slice because
	// callback() modifies the slice
	return dag, &executed
}

func assertBuild(
	t *testing.T,
	dag *DAG,
	expect []buildexpect,
	executed []string) {
	assert.Equal(t, len(expect), len(executed),
		"expected %d build attempts (%v), but got %d",
		len(expect), expect, len(executed))
	for i, expect := range expect {
		assert.Equal(t, expect.name, executed[i],
			"action %d: expected %s", i, expect.name)
		actualstate := dag.lookup(expect.name).State()
		assert.Equal(t, expect.state, actualstate,
			"target: %s (state = %v)", expect.name, actualstate)
	}
}

type buildexpect struct {
	name  string
	state NodeState
}

func Test_BuildError_Error(t *testing.T) {
	dag := NewDAG()
	err := &BuildError{}
	err.attempts = 43
	err.failed = mknodelist(
		dag, "foo", "bar", "baz")
	assert.Equal(t,
		"failed to build 3 of 43 targets: foo, bar, baz", err.Error())
	err.attempts = -1
	assert.Equal(t,
		"failed to build target: foo", err.Error())

	err.attempts = 17
	err.failed = mknodelist(
		dag, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k")
	assert.Equal(t,
		"failed to build 11 of 17 targets: a, b, c, d, e, f, g, h, i, ...",
		err.Error())
}

func Test_joinNodes(t *testing.T) {
	dag := NewDAG()
	nodes := mknodelist(dag, "blargh", "merp", "whoosh", "fwob", "whee")

	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 10, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 5, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, ...", joinNodes(", ", 4, nodes))
	assert.Equal(t,
		"blargh!*!merp!*!...", joinNodes("!*!", 3, nodes))
}

func mknodelist(dag *DAG, names ...string) []Node {
	result := make([]Node, len(names))
	for i, name := range names {
		result[i] = MakeStubNode(dag, name)
	}
	return result
}
