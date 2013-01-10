// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"fmt"
	"os"
	"strings"

	"fubsy/dag"
	"fubsy/db"
	"fubsy/log"
)

type stateset map[dag.NodeState]bool

type BuildState struct {
	graph        *dag.DAG
	db           BuildDB
	options      BuildOptions
	changestates stateset
}

// user options, typically from the command line
type BuildOptions struct {
	// keep building even after one target fails (default: stop on
	// first failure)
	KeepGoing bool

	// pessimistically assume that someone might have modified
	// intermediate targets behind our back (default: check only
	// original sources)
	CheckAll bool
}

type BuildError struct {
	// nodes that failed to build
	failed []dag.Node

	// total number of nodes that we attempted to build
	attempts int
}

func NewBuildState(graph *dag.DAG, db BuildDB, options BuildOptions) *BuildState {
	return &BuildState{graph: graph, db: db, options: options}
}

// The heart of Fubsy: do a depth-first walk of the dependency graph
// to discover nodes in topological order, then (re)build nodes that
// are stale or missing. Skip target nodes that are "tainted" by
// upstream failure. Returns a single error object summarizing what
// (if anything) went wrong; error details are reported "live" as
// builds fail (e.g. to the console or a GUI window) so the user gets
// timely feedback.
func (self *BuildState) BuildTargets(targets *dag.NodeSet) error {
	// What sort of nodes do we check for changes?
	self.setChangeStates()
	log.Debug("build", "building %d targets", targets.Length())

	builderr := new(BuildError)
	visit := func(node dag.Node) error {
		if node.State() == dag.SOURCE {
			// can't build original source nodes!
			return nil
		}
		if node.BuildRule() == nil {
			panic("node is a target, but has no build rule: " + node.Name())
		}

		checkInitialState(node)

		// do we need to build this node? can we?
		build, tainted, err := self.considerNode(node)
		log.Debug("build", "node %s: build=%v, tainted=%v err=%v\n",
			node, build, tainted, err)
		if err != nil {
			return err
		}

		if tainted {
			node.SetState(dag.TAINTED)
		} else if build {
			ok := self.buildNode(node, builderr)
			if !ok && !self.keepGoing() {
				// attempts counter is not very useful when we break
				// out of the build early
				builderr.attempts = -1
				return builderr
			}
			if ok {
				self.recordNode(node)
			}
		}
		return nil
	}
	err := self.graph.DFS(targets, visit)
	if err == nil && len(builderr.failed) > 0 {
		// build failures in keep-going mode
		err = builderr
	}
	return err
}

func (self *BuildState) setChangeStates() {
	// Default is like tup: only check original source nodes and nodes
	// that have just been built.
	changestates := make(stateset)
	changestates[dag.SOURCE] = true
	changestates[dag.BUILT] = true
	if self.checkAll() {
		// Optional SCons-like behaviour: assume the user has been
		// sneaking around and modifying .o or .class files behind our
		// back and check everything. (N.B. this is unnecessary if the
		// user sneakily *removes* intermediate targets; that case
		// should be handled just fine by default.)
		changestates[dag.UNKNOWN] = true
	}
	self.changestates = changestates
}

// panic if node is in an impossible state for starting its visit
func checkInitialState(node dag.Node) {
	if node.State() != dag.UNKNOWN {
		// we just skipped SOURCE nodes, the other states are only set
		// while visiting a node, and we should only visit each node
		// once
		panic(fmt.Sprintf(
			"visiting node %v, state = %s (should be UNKNOWN)",
			node, node.State()))
	}
}

// Inspect node and its parents to see if we need to build it. Return
// build=true if we should build it, tainted=true if we should skip
// building this node due to upstream failure. Return non-nil err if
// there were unexpected node errors (error checking existence or
// change status).
func (self *BuildState) considerNode(node dag.Node) (
	build bool, tainted bool, err error) {

	var exists, changed bool
	exists, err = node.Exists() // obvious rebuild (unless tainted)
	if err != nil {
		return
	}
	missing := !exists

	var record *db.BuildRecord
	if !missing {
		// skip DB lookup for missing nodes: the only thing that will
		// stop us from rebuilding them is a failed parent, and that
		// check comes later
		record, err = self.db.LookupNode(node.Name())
		if err != nil {
			return
		}
		if record != nil {
			log.Debug("build", "old parents of %s:", node)
			log.DebugDump("build", record)
		}
	}

	build = missing
	parents := self.graph.ParentNodes(node)

	// Check if any of node's former parents have been removed.
	if !build && record != nil {
		build = parentsRemoved(parents, record)
	}

	for _, parent := range parents {
		pstate := parent.State()
		if pstate == dag.FAILED || pstate == dag.TAINTED {
			build = false
			tainted = true
			return // no further inspection required
		}

		var oldsig []byte
		if record != nil {
			oldsig = record.SourceSignature(parent.Name())
			if oldsig == nil {
				// New parent for this node: rebuild unless another
				// parent is failed/tainted.
				build = true
			}
		}

		if build {
			continue
		}
		changed, err = self.parentChanged(parent, pstate, oldsig)
		if err != nil {
			return
		}
		if changed {
			// Do NOT return here: we need to continue inspecting
			// parents to make sure they don't taint this node with
			// upstream failure.
			build = true
		}
	}
	return
}

func parentsRemoved(parents []dag.Node, record *db.BuildRecord) bool {
	parentset := make(map[string]bool)
	for _, parent := range parents {
		parentset[parent.Name()] = true
	}
	for _, name := range record.Parents() {
		if !parentset[name] {
			return true
		}
	}
	return false
}

func (self *BuildState) parentChanged(
	parent dag.Node, pstate dag.NodeState, oldsig []byte) (
	bool, error) {

	var cursig []byte
	var err error
	if !(pstate == dag.SOURCE || pstate == dag.BUILT || self.checkAll()) {
		// parent is an intermediate target that has not been built in
		// the current run. We don't want the expense of calling
		// Signature() for target nodes, because people don't normally
		// modify targets behind their build tool's back. But it might
		// have been modified by a previous build, in which case we'll
		// use the signature previously recorded for parent *as a
		// target*. This can still be fooled by modifying intermediate
		// targets behind Fubsy's back, but that's what --check-all is
		// for.
		precord, err := self.db.LookupNode(parent.Name())
		if err != nil {
			return false, err
		}
		if precord != nil {
			cursig = precord.TargetSignature()
		}
	}

	if cursig == nil {
		cursig, err = parent.Signature()
		if err != nil {
			// This should not happen: parent should exist and be readable,
			// since we've already visited it earlier in the build and we
			// avoid looking at failed/tainted parents.
			return false, err
		}
	}
	changed := parent.Changed(cursig, oldsig)
	//log.Verbose("parent %s: oldsig=%v, cursig=%v, changed=%v",
	//	parent, oldsig, cursig, changed)
	return changed, nil
}

// Build the specified node (caller has determined that it should be
// built and can be built). On failure, report the error (e.g. to the
// console, a GUI window, ...) and return false. On success, return
// true.
func (self *BuildState) buildNode(node dag.Node, builderr *BuildError) bool {
	rule := node.BuildRule()
	log.Verbose("building node %s, action=%s\n", node, rule.ActionString())
	node.SetState(dag.BUILDING)
	builderr.attempts++
	targets, errs := rule.Execute()
	if len(errs) > 0 {
		// Normal, everyday build failure: report the precise problem
		// immediately, and accumulate summary info in the caller.
		for _, tnode := range targets {
			tnode.SetState(dag.FAILED)
		}
		self.reportFailure(errs)
		builderr.addFailure(node)
		return false
	}
	for _, tnode := range targets {
		tnode.SetState(dag.BUILT)
	}
	return true
}

func (self *BuildState) reportFailure(errs []error) {
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "build failure: %s\n", err)
	}
}

func (self *BuildState) recordNode(node dag.Node) error {
	sig, err := node.Signature()
	if err != nil {
		return err
	}

	record := db.NewBuildRecord()
	record.SetTargetSignature(sig)
	for _, parent := range self.graph.ParentNodes(node) {
		sig, err = parent.Signature()
		if err != nil {
			return err
		}
		record.AddParent(parent.Name(), sig)
	}
	err = self.db.WriteNode(node.Name(), record)
	if err != nil {
		return err
	}
	return nil
}

func (self *BuildState) keepGoing() bool {
	return self.options.KeepGoing
}

func (self *BuildState) checkAll() bool {
	return self.options.CheckAll
}

func (self *BuildError) addFailure(node dag.Node) {
	self.failed = append(self.failed, node)
}

func (self *BuildError) Error() string {
	if len(self.failed) == 0 {
		panic("called Error() on a BuildError with no failures: " +
			"there is no error here!")
	}

	if self.attempts > 0 {
		failed := joinNodes(", ", 10, self.failed)
		return fmt.Sprintf(
			"failed to build %d of %d targets: %s",
			len(self.failed), self.attempts, failed)
	}
	return fmt.Sprintf("failed to build target: %s", self.failed[0])
}

func joinNodes(delim string, max int, nodes []dag.Node) string {
	if len(nodes) < max {
		max = len(nodes)
	}
	svalues := make([]string, max)
	for i := 0; i < max; i++ {
		svalues[i] = nodes[i].String()
	}
	if len(nodes) > max {
		svalues[max-1] = "..."
	}
	return strings.Join(svalues, delim)
}
