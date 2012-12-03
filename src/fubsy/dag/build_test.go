package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
)

func Test_BuildState_findStaleTargets(t *testing.T) {
	dag := makeSimpleGraph()

	setup := func(goalid ...int) (*DAG, *BuildState) {
		goal := bit.New(goalid...)
		relevant := dag.FindRelevantNodes(NodeSet(goal))
		rdag, errors := dag.Rebuild(relevant)
		assert.Equal(t, 0, len(errors))
		rdag.ComputeChildren()
		bstate := rdag.NewBuildState()
		return rdag, bstate
	}

	test := func(rdag *DAG, bstate *BuildState, expect []string) {
		stale, errors := bstate.findStaleTargets()
		assert.Equal(t, 0, len(errors))
		names := setToNames(rdag, stale)
		assert.Equal(t, expect, names)
	}

	// goal = {tool1, tool2}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c, tool2.c}
	// initial stale = {tool1.o, misc.o, util.o, tool2.o}
	rdag, bstate := setup(0, 1)
	expect := []string {"tool1.o", "misc.o", "util.o", "tool2.o"}
	test(rdag, bstate, expect)

	// goal = {tool1}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	// initial stale = {tool1.o, misc.o, util.o}
	rdag, bstate = setup(0)
	expect = []string {"tool1.o", "misc.o", "util.o"}
	test(rdag, bstate, expect)

	// goal = {tool2}
	// sources = {tool2.c, util.h, util.c}
	// initial stale = {tool2.o, util.o}
	rdag, bstate = setup(1)
	expect = []string {"util.o", "tool2.o"}
	test(rdag, bstate, expect)
}

func Test_joinNodes(t *testing.T) {
	dag := NewDAG()
	nodes := []Node {
		makestubnode(dag, "blargh"),
		makestubnode(dag, "merp"),
		makestubnode(dag, "whoosh"),
		makestubnode(dag, "fwob"),
		makestubnode(dag, "whee"),
	}

	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 10, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, fwob, whee", joinNodes(", ", 5, nodes))
	assert.Equal(t,
		"blargh, merp, whoosh, ...", joinNodes(", ", 4, nodes))
	assert.Equal(t,
		"blargh!*!merp!*!...", joinNodes("!*!", 3, nodes))
}
