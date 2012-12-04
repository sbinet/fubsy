package dag

import (
	"testing"
	//"fmt"
	"errors"
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

// full build (all targets), all actions succeed
func Test_BuildState_BuildStaleTargets_full_success(t *testing.T) {
	dag, executed := setupBuild()

	expect := []buildexpect {
		// initial stale set: children of original source nodes, ordered by
		// node ID
		{"tool1.o", BUILT},
		{"misc.o",  BUILT},
		{"util.o",  BUILT},
		{"tool2.o", BUILT},

		// second pass: as we build each of the above, we remove it from
		// the stale set and put its children in the stale set, then
		// iterate again over the stale set in node ID order
		{"tool1",  BUILT},
		{"tool2",  BUILT},
	}

	bstate := dag.NewBuildState()
	err := bstate.BuildStaleTargets()
	assert.Nil(t, err)
	assertBuild(t, dag, expect, *executed)

	// hmmmm: should we have states DIRTY and CLEAN that only apply to
	// original source nodes, determined by result of Changed()? seems
	// redundant since Changed() is a method of the Node, so the Node
	// should know whether it is clean or dirty
	assert.Equal(t, UNKNOWN, dag.lookup("tool2.c").State())
	assert.Equal(t, UNKNOWN, dag.lookup("misc.h").State())
	assert.Equal(t, UNKNOWN, dag.lookup("util.c").State())
}

// full build (all targets), one action fails
func Test_BuildState_BuildStaleTargets_full_failure(t *testing.T) {
	dag, executed := setupBuild()

	// fail to building misc.{c,h} -> misc.o: that will block building
	// tool1, but not tool2 (since keepGoing() always returns true)
	action := dag.lookup("misc.o").Action().(*stubaction)
	action.ok = false

	expect := []buildexpect {
		// initial stale set: children of original source nodes, ordered by
		// node ID
		{"tool1.o", BUILT},
		{"misc.o",  FAILED},
		{"util.o",  BUILT},
		{"tool2.o", BUILT},

		// second pass: as we build each of the above, we remove it from
		// the stale set and put its children in the stale set, then
		// iterate again over the stale set in node ID order
		{"tool2",  BUILT},
	}

	bstate := dag.NewBuildState()
	err := bstate.BuildStaleTargets()
	assert.NotNil(t, err)
	assertBuild(t, dag, expect, *executed)

	// we don't even try to build tool1, since one of its parents failed
	assert.Equal(t, TAINTED, dag.lookup("tool1").State())
}

func setupBuild() (*DAG, *[]string) {
	dag := makeSimpleGraph()
	dag.ComputeChildren()

	// add stub actions to every target node, so we know when each
	// action's Execute() method is called
	executed := []string {}
	callback := func(desc string) {
		executed = append(executed, desc)
	}
	for id, node := range dag.nodes {
		if !dag.parents[id].IsEmpty() {
			action := newstubaction("build " + node.Name(), callback, true)
			node.SetAction(action)
		}
	}
	// need to return a pointer to the executed slice because
	// callback() modifies the slice
	return dag, &executed
}

func assertBuild(
	t *testing.T,
	dag *DAG,
	expect []buildexpect,
	executed []string) {
	assert.Equal(t, len(expect), len(executed))
	for i, expect := range expect {
		desc := "build " + expect.name
		assert.Equal(t, desc, executed[i],
			"action %d: expected %s", i, expect.name)
		actualstate := dag.lookup(expect.name).State()
		assert.Equal(t, expect.state, actualstate,
			"target: %s (state = %v)", expect.name, actualstate)
	}
}

type buildexpect struct {
	name string
	state NodeState
}

type stubaction struct {
	desc string

	// takes desc -- used for recording order in which targets are built
	callback func(string)

	// true if this action should succeed
	ok bool
}

func newstubaction(desc string, callback func(string), ok bool) *stubaction {
	return &stubaction{desc, callback, ok}
}

func (self stubaction) String() string {
	return self.desc
}

func (self stubaction) Execute() error {
	self.callback(self.desc)
	if !self.ok {
		return errors.New("action failed")
	}
	return nil
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
