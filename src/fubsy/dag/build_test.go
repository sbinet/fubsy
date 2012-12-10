package dag

import (
	"testing"
	//"fmt"
	"errors"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
	"fubsy/types"
)

// full build (all targets), all actions succeed
func Test_BuildState_BuildTargets_full_success(t *testing.T) {
	dag, executed := setupBuild()

	expect := []buildexpect {
		{"tool1.o", BUILT},
		{"misc.o",  BUILT},
		{"util.o",  BUILT},
		{"tool1",   BUILT},
		{"tool2.o", BUILT},
		{"tool2",   BUILT},
	}

	bstate := dag.NewBuildState()
	ns := types.NewValueStack()
	goal := NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(&ns, goal)
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
	action := dag.lookup("misc.o").Action().(*stubaction)
	action.ok = false

	expect := []buildexpect {
		{"tool1.o", BUILT},
		{"misc.o",  FAILED},
		{"util.o",  BUILT},
		{"tool2.o", BUILT},
		{"tool2",   BUILT},
	}

	bstate := dag.NewBuildState()
	ns := types.NewValueStack()
	goal := NodeSet(bit.New(0, 1))
	err := bstate.BuildTargets(&ns, goal)
	assert.NotNil(t, err)
	assertBuild(t, dag, expect, *executed)

	// we don't even try to build tool1, since one of its parents failed
	assert.Equal(t, TAINTED, dag.lookup("tool1").State())
}

func setupBuild() (*DAG, *[]string) {
	dag := makeSimpleGraph()

	// add a stub action to every target node, so we know when each
	// action's Execute() method is called
	executed := []string {}
	callback := func(name string) {
		executed = append(executed, name)
	}
	for id, node := range dag.nodes {
		if !dag.parents[id].IsEmpty() {
			action := newstubaction(node.Name(), callback, true)
			node.SetAction(action)
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

func (self stubaction) Execute(ns types.Namespace) error {
	self.callback(self.desc)
	if !self.ok {
		return errors.New("action failed")
	}
	return nil
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
		dag, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j" ,"k")
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
		result[i] = makestubnode(dag, name)
	}
	return result
}
