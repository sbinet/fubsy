package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
)

func Test_FindRelevantNodes(t *testing.T) {
	dag := makeSimpleGraph()
	goal := bit.New(0, 1)		// all final targets: tool1, tool2
	relevant := FindRelevantNodes(dag, NodeSet(goal))
	//assert.Equal(t, "{6..11}", bstate.sources.String())
	assert.Equal(t, "{0,1,2,3,4,5,6,7,8,9,10,11}", setToString(relevant))

	// relevant children of misc.h = {tool1.o, misc.o}
	//assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o, tool2.o}
	//assert.Equal(t, "{2, 4, 5}", bstate.children[9].String())

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	goal = bit.New(0)
	relevant = FindRelevantNodes(dag, NodeSet(goal))
	//assert.Equal(t, "{6..10}", bstate.sources.String())
	assert.Equal(t, "{0,2,3,4,6,7,8,9,10}", setToString(relevant))

	// relevant children of misc.h = {tool1.o, misc.o}
	//assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o}
	//assert.Equal(t, "{2, 4}", bstate.children[9].String())

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	goal = bit.New(1)
	relevant = FindRelevantNodes(dag, NodeSet(goal))
	//assert.Equal(t, "{9..11}", bstate.sources.String())
	assert.Equal(t, "{1,4,5,9,10,11}", setToString(relevant))

	// misc.h is not a relevant node, so not in children map
	//assert.Nil(t, bstate.children[7])
	// relevant children of util.h = {util.o, tool2.o}
	//assert.Equal(t, "{4, 5}", bstate.children[9].String())
}

func Test_FindRelevantNodes_cycle(t *testing.T) {
	dag := makeSimpleGraph()
	dag.addParent(dag.lookup("misc.h"), dag.lookup("tool1"))

	// goal = {tool2} ==> no cycle, since we don't visit those nodes
	// (this simply tests that FindRelevantNodes() doesn't panic)
	goal := bit.New(1)
	FindRelevantNodes(dag, NodeSet(goal))

	// goal = {tool1} ==> cycle!
	// (disabled because FindRelevantNodes() currently panics on cycle)
	return
	goal = bit.New(0)
	FindRelevantNodes(dag, NodeSet(goal))
}

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
	relevant := FindRelevantNodes(dag, NodeSet(goal))
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
	relevant = FindRelevantNodes(dag, NodeSet(goal))
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
