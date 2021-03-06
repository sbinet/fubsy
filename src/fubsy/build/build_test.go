// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
	"fubsy/db"
	//"fubsy/log"
)

// full build of all targets, all actions succeed
func Test_BuildState_BuildTargets_all_changed(t *testing.T) {
	// This is actually an unusual case: we rebuild all targets
	// because all sources have changed. A much more likely reason for
	// a full build is that all targets are missing, e.g. a fresh
	// working dir.
	oldsig := []byte{0}
	newsig := []byte{1}
	graph, executed := setupBuild(true, newsig)
	db := makeFakeDB(graph, oldsig)

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.BUILT},
		{"util.o", dag.BUILT},
		{"tool1", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	bstate := NewBuildState(graph, db, BuildOptions{})
	goal := graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)

	assert.Equal(t, dag.SOURCE, graph.Lookup("tool2.c").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("misc.h").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("util.c").State())
}

// full build because all targets are missing (much more realistic)
func Test_BuildState_BuildTargets_all_missing(t *testing.T) {
	sig := []byte{0}
	graph, executed := setupBuild(false, sig)
	db := makeFakeDB(graph, sig)

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.BUILT},
		{"util.o", dag.BUILT},
		{"tool1", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	bstate := NewBuildState(graph, db, BuildOptions{})
	goal := graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)

	assert.Equal(t, dag.SOURCE, graph.Lookup("tool2.c").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("misc.h").State())
	assert.Equal(t, dag.SOURCE, graph.Lookup("util.c").State())
}

// all targets exist, but the BuildDB is empty: rebuild everything
// because we have no way of knowing what changed
func Test_BuildState_BuildTargets_empty_db(t *testing.T) {
	sig := []byte{0}
	graph, executed := setupBuild(true, sig)
	bdb := db.NewFakeDB()

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.BUILT},
		{"util.o", dag.BUILT},
		{"tool1", dag.BUILT},
		{"tool2.o", dag.BUILT},
		{"tool2", dag.BUILT},
	}

	bstate := NewBuildState(graph, bdb, BuildOptions{})
	goal := graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)
}

// full successful build, then try some incremental rebuilds
func Test_BuildState_BuildTargets_rebuild(t *testing.T) {
	sig := []byte{0}
	db, goal, opts := fullBuild(t, sig)

	// now the rebuild, after marking all nodes unchanged
	graph, executed := setupBuild(true, sig)

	expect := []buildexpect{}
	bstate := NewBuildState(graph, db, opts)
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)

	// again, but this time change one source file (misc.h, forcing
	// rebuilds of misc.o and tool1.o -- but those two will appear
	// unchanged, so we short-circuit the build and do *not* rebuild
	// tool1)
	graph, executed = setupBuild(true, sig)
	graph.Lookup("misc.h").(*dag.StubNode).SetSignature([]byte{1})

	expect = []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.BUILT},
	}
	bstate = NewBuildState(graph, db, opts)
	err = bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)
}

// full build (all targets), one action fails
func Test_BuildState_BuildTargets_one_failure(t *testing.T) {
	sig := []byte{0}
	graph, executed := setupBuild(false, sig)
	db := makeFakeDB(graph, sig)

	// fail to build misc.{c,h} -> misc.o: that will stop the build
	rule := graph.Lookup("misc.o").BuildRule().(*dag.StubRule)
	rule.SetFail(true)

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"misc.o", dag.FAILED},
	}

	opts := BuildOptions{}
	bstate := NewBuildState(graph, db, opts)
	goal := graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, graph, expect, *executed)

	// we didn't even think about building util.o, tool1, etc: an
	// earlier node failed and the build terminates on first failure
	assert.Equal(t, dag.UNKNOWN, graph.Lookup("util.o").State())
	assert.Equal(t, dag.UNKNOWN, graph.Lookup("tool1").State())
	assert.Equal(t, dag.UNKNOWN, graph.Lookup("tool2").State())
}

// full build (all targets), one action fails, --keep-going true
func Test_BuildState_BuildTargets_one_failure_keep_going(t *testing.T) {
	// this is the same as the previous test except that
	// opts.KeepGoing == true: we don't terminate the build on first
	// failure, but carry on and consider building tool1, then mark it
	// TAINTED because one of its ancestors (misc.o) failed to build

	sig := []byte{0}
	graph, executed := setupBuild(false, sig)
	db := makeFakeDB(graph, sig)

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
	bstate := NewBuildState(graph, db, opts)
	goal := graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.NotNil(t, err)
	assertBuild(t, graph, expect, *executed)

	assert.Equal(t, dag.TAINTED, graph.Lookup("tool1").State())
}

func Test_BuildState_BuildTargets_add_source(t *testing.T) {
	// do a full build (all targets missing), then add one source file
	// and ensure that downstream targets are rebuilt
	sig := []byte{0}
	db, goal, opts := fullBuild(t, sig)

	// feep.h is the new file, a parent of tool1.o: we will rebuild
	// tool1.o and tool1 (after ensuring that tool1.o is changed by
	// the rebuild)
	graph, executed := setupBuild(true, sig)
	newnode := dag.MakeStubNode(graph, "feep.h")
	newnode.SetState(dag.SOURCE)
	child := graph.Lookup("tool1.o").(*dag.StubNode)
	graph.AddParent(child, newnode)
	child.SetSignature([]byte{1})

	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"tool1", dag.BUILT}}
	bstate := NewBuildState(graph, db, opts)
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)
}

func Test_BuildState_BuildTargets_remove_source(t *testing.T) {
	// full build, then remove one source file and ensure that
	// downstream targets (anything that formerly depended on the
	// removed file) are rebuilt
	sig := []byte{0}
	db, goal, opts := fullBuild(t, sig)

	// same as setupBuild() does, but without misc.{c,h,o}
	graph := makeSmallerGraph()
	setNodeExists(graph, true) // all exist
	setNodeSigs(graph, sig)    // none changed
	executed := addTrackingRules(graph)
	graph.MarkSources()

	// now build with the smaller graph: we should rebuild tool.o
	// (formerly depended on misc.h) and tool1 (formerly depended on
	// misc.o)
	expect := []buildexpect{
		{"tool1.o", dag.BUILT},
		{"tool1", dag.BUILT},
	}
	bstate := NewBuildState(graph, db, opts)
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)
}

func Test_BuildState_BuildTargets_modify_shared_source(t *testing.T) {
	// Do a full build, then modify a shared source file (util.c).
	// Rebuild with goal={tool1} and ensure stuff is rebuilt
	// correctly. Then do a third build with goal={tool2} and ensure
	// that it too is rebuilt because of the change to util.{c,o}.
	oldsig := []byte{0}
	db, goal, opts := fullBuild(t, oldsig)

	newsig := []byte{1}
	graph, executed := setupBuild(true, oldsig)
	graph.Lookup("util.c").(*dag.StubNode).SetSignature(newsig)
	graph.Lookup("util.o").(*dag.StubNode).SetSignature(newsig)

	// second build (tool1)
	goal = graph.MakeNodeSet("tool1")
	expect := []buildexpect{
		{"util.o", dag.BUILT},
		{"tool1", dag.BUILT},
	}
	bstate := NewBuildState(graph, db, opts)
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)

	// third build (tool2)
	graph, executed = setupBuild(true, oldsig)
	graph.Lookup("util.c").(*dag.StubNode).SetSignature(newsig)
	graph.Lookup("util.o").(*dag.StubNode).SetSignature(newsig)
	goal = graph.MakeNodeSet("tool2")
	expect = []buildexpect{
		{"tool2", dag.BUILT},
	}
	bstate = NewBuildState(graph, db, opts)
	err = bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assertBuild(t, graph, expect, *executed)
}

func Test_BuildState_BuildTargets_unsignable_target(t *testing.T) {
	// if we run an action that is supposed to build a target, but the
	// target is unreadable (doesn't exist, permission denied, ...),
	// then the build fails immediately
	snode := dag.NewStubNode("source")
	tnode := NewUnsignableNode("target")
	graph := dag.NewDAG()
	graph.AddNode(snode)
	graph.AddNode(tnode)
	graph.AddParent(tnode, snode)
	executed := addTrackingRules(graph)
	graph.MarkSources()

	sig := []byte{0}
	db := makeFakeDB(graph, sig)
	opts := BuildOptions{}
	bstate := NewBuildState(graph, db, opts)

	// target gets built; the build fails after that, calculating its
	// signature
	goal := graph.MakeNodeSet("target")
	expect := []buildexpect{
		{"target", dag.BUILT},
	}
	err := bstate.BuildTargets(goal)
	assert.Equal(t,
		"could not compute signature of target \"target\": nah", err.Error())
	assertBuild(t, graph, expect, *executed)
}

func setupBuild(exists bool, sig []byte) (*dag.DAG, *[]string) {
	graph := makeSimpleGraph()
	setNodeExists(graph, exists)
	setNodeSigs(graph, sig)
	executed := addTrackingRules(graph)
	graph.MarkSources()
	return graph, executed
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

func makeSmallerGraph() *dag.DAG {
	// construct the same graph as makeSimpleGraph(), but minus
	// misc.{h,c}
	tdag := dag.NewTestDAG()
	tdag.Add("tool1", "tool1.o", "util.o")
	tdag.Add("tool2", "tool2.o", "util.o")
	tdag.Add("tool1.o", "tool1.c", "util.h")
	tdag.Add("util.o", "util.h", "util.c")
	tdag.Add("tool2.o", "tool2.c", "util.h")
	tdag.Add("tool1.c")
	tdag.Add("util.h")
	tdag.Add("util.c")
	tdag.Add("tool2.c")
	return tdag.Finish()
}

func setNodeExists(graph *dag.DAG, exists bool) {
	for _, node := range graph.Nodes() {
		node.(*dag.StubNode).SetExists(exists)
	}
}

func setNodeSigs(graph *dag.DAG, sig []byte) {
	for _, node := range graph.Nodes() {
		node.(*dag.StubNode).SetSignature(sig)
	}
}

// add a stub build rule to every target node, so we track when each
// rule's Execute() method is called
func addTrackingRules(graph *dag.DAG) *[]string {
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
	// need to return a pointer to the executed slice because
	// callback() modifies the slice
	return &executed
}

func makeFakeDB(graph *dag.DAG, sig []byte) BuildDB {
	var bdb BuildDB = db.NewFakeDB()
	for _, node := range graph.Nodes() {
		record := db.NewBuildRecord()
		record.SetTargetSignature(sig)
		for _, parent := range graph.ParentNodes(node) {
			record.AddParent(parent.Name(), sig)
		}
		bdb.WriteNode(node.Name(), record)
	}
	return bdb
}

func fullBuild(t *testing.T, sig []byte) (
	bdb BuildDB, goal *dag.NodeSet, opts BuildOptions) {
	graph, executed := setupBuild(false, sig)
	bdb = makeFakeDB(graph, sig)
	opts = BuildOptions{}
	bstate := NewBuildState(graph, bdb, opts)
	goal = graph.MakeNodeSet("tool1", "tool2")
	err := bstate.BuildTargets(goal)
	assert.Nil(t, err)
	assert.Equal(t, 6, len(*executed))
	return
}

func assertBuild(
	t *testing.T,
	graph *dag.DAG,
	expect []buildexpect,
	executed []string) {

	actual := make([]buildexpect, len(executed))
	for i, name := range executed {
		state := graph.Lookup(name).State()
		actual[i] = buildexpect{name, state}
	}

	if len(expect) != len(actual) {
		t.Errorf("expected %d build attempts, but got %d\n"+
			"expect: %v\n"+
			"actual: %v",
			len(expect), len(actual), expect, actual)
		return
	}
	for i := range expect {
		assert.Equal(t, expect[i], actual[i])
	}
}

type buildexpect struct {
	name  string
	state dag.NodeState
}

func (self buildexpect) GoString() string {
	return self.String()
}

func (self buildexpect) String() string {
	return fmt.Sprintf("(%s, %s)", self.name, self.state)
}

func Test_BuildError_Error(t *testing.T) {
	graph := dag.NewDAG()
	err := &BuildError{}
	err.attempts = 43
	err.failed = mknodelist(
		graph, "foo", "bar", "baz")
	assert.Equal(t,
		`failed to build 3 of 43 targets: "foo", "bar", "baz"`, err.Error())
	err.attempts = -1
	assert.Equal(t,
		`failed to build target: "foo"`, err.Error())

	err.attempts = 17
	err.failed = mknodelist(
		graph, "a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k")
	assert.Equal(t,
		`failed to build 11 of 17 targets: "a", "b", "c", "d", "e", "f", "g", "h", "i", ...`,
		err.Error())
}

func Test_joinNodes(t *testing.T) {
	graph := dag.NewDAG()
	nodes := mknodelist(graph, "blargh", "merp", "whoosh", "fwob", "whee")

	assert.Equal(t,
		`"blargh", "merp", "whoosh", "fwob", "whee"`,
		joinNodes(", ", 10, nodes))
	assert.Equal(t,
		`"blargh", "merp", "whoosh", "fwob", "whee"`,
		joinNodes(", ", 5, nodes))
	assert.Equal(t,
		`"blargh", "merp", "whoosh", ...`,
		joinNodes(", ", 4, nodes))
	assert.Equal(t,
		`"blargh"!*!"merp"!*!...`,
		joinNodes("!*!", 3, nodes))
}

func mknodelist(graph *dag.DAG, names ...string) []dag.Node {
	result := make([]dag.Node, len(names))
	for i, name := range names {
		result[i] = dag.MakeStubNode(graph, name)
	}
	return result
}

type UnsignableNode struct {
	dag.StubNode
}

func NewUnsignableNode(name string) *UnsignableNode {
	return &UnsignableNode{StubNode: *dag.NewStubNode(name)}
}

func (self *UnsignableNode) Signature() ([]byte, error) {
	return nil, errors.New("nah")
}
