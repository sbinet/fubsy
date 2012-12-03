package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
)

func Test_BuildState_FindStaleTargets(t *testing.T) {
	dag := makeSimpleGraph()
	dag.ComputeChildren()
	//sources := newNodeSet(

	//	dag, "tool1.c", "misc.h", "misc.c", "util.h", "util.c")

	// goal = {tool1, tool2}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c, tool2.c}
	// initial rebuild = {tool1.o, misc.o, util.o, tool2.o}
	//goal = bit.New(0, 1)
	//relevant = FindRelevantNodes(dag, NodeSet(goal))
	//sources = newNodeSet(
	//	dag, "tool1.c", "misc.h", "misc.c", "util.h", "util.c", "tool2.c")
	expect := []string {"tool1.o", "misc.o", "util.o", "tool2.o"}
	stale, errors := FindStaleTargets(dag)
	assert.Equal(t, 0, len(errors))
	names := setToNames(dag, stale)
	assert.Equal(t, expect, names)

	// goal = {tool1}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	// initial rebuild = {tool1.o, misc.o, util.o}
	goal := bit.New(0)
	relevant := dag.FindRelevantNodes(NodeSet(goal))
	rdag, errors := dag.Rebuild(relevant)
	assert.Equal(t, 0, len(errors))
	rdag.ComputeChildren()

	expect = []string {"tool1.o", "misc.o", "util.o"}
	stale, errors = FindStaleTargets(rdag)
	names = setToNames(rdag, stale)
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, expect, names)
	//assert.Equal(t, "{2,3,4}", setToString(stale))

	// goal = {tool2}
	// sources = {tool2.c, util.h, util.c}
	// initial rebuild = {tool2.o, util.o}
	goal = bit.New(1)
	relevant = dag.FindRelevantNodes(NodeSet(goal))
	rdag, errors = dag.Rebuild(relevant)
	assert.Equal(t, 0, len(errors))
	rdag.ComputeChildren()

	expect = []string {"util.o", "tool2.o"}
	stale, errors = FindStaleTargets(rdag)
	names = setToNames(rdag, stale)
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, expect, names)
	//assert.Equal(t, "{4,5}", setToString(stale))
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
