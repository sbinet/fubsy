// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package build

import (
	"fmt"
	"os"
	"strings"

	"fubsy/dag"
	"fubsy/log"
)

type BuildState struct {
	graph   *dag.DAG
	options BuildOptions
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

func NewBuildState(graph *dag.DAG, options BuildOptions) *BuildState {
	return &BuildState{graph: graph, options: options}
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
	changestates := self.getChangeStates()
	log.Debug("build", "building %d targets", targets.Length())

	builderr := new(BuildError)
	visit := func(node dag.Node) error {
		if node.State() == dag.SOURCE {
			// can't build original source nodes!
			return nil
		}

		checkInitialState(node)

		// do we need to build this node? can we?
		missing, stale, tainted, err :=
			self.considerNode(changestates, node)
		log.Debug("build",
			"node %s: missing=%v, stale=%v, tainted=%v, err=%v\n",
			node, missing, stale, tainted, err)
		if err != nil {
			return err
		}

		if tainted {
			node.SetState(dag.TAINTED)
		} else if missing || stale {
			ok := self.buildNode(node, builderr)
			if !ok && !self.keepGoing() {
				// attempts counter is not very useful when we break
				// out of the build early
				builderr.attempts = -1
				return builderr
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

func (self *BuildState) getChangeStates() map[dag.NodeState]bool {
	// Default is like tup: only check original source nodes and nodes
	// that have just been built.
	changestates := make(map[dag.NodeState]bool)
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
	return changestates
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
// tainted=true if we should skip building this node due to upstream
// failure. Return stale=true if we should build this node because at
// least one of its parents has changed. Return missing=true if we
// should build this node because its resource is missing. Return
// non-nil err if there were unexpected node errors (error checking
// existence or change status).
func (self *BuildState) considerNode(
	changestates map[dag.NodeState]bool, node dag.Node) (
	missing, stale, tainted bool, err error) {

	var exists, changed bool
	exists, err = node.Exists() // obvious rebuild (unless tainted)
	if err != nil {
		return
	}
	missing = !exists
	stale = false   // parents have changed
	tainted = false // failures upstream: do not rebuild

	parentnodes := self.graph.ParentNodes(node)
	for _, parent := range parentnodes {
		pstate := parent.State()
		if pstate == dag.FAILED || pstate == dag.TAINTED {
			tainted = true
			return // no further inspection required
		}

		// Try *really hard* to avoid calling parent.Changed(), because
		// it's likely to do I/O: prone to fail, slow, etc.
		if missing || stale || !changestates[pstate] {
			continue
		}
		changed, err = parent.Changed()
		if err != nil {
			// This should not happen: parent should exist and be readable,
			// since we've already visited it earlier in the build and we
			// avoid looking at failed/tainted parents.
			return
		}
		if changed {
			stale = true
			// Do NOT return here: we need to continue inspecting parents
			// to make sure they don't taint this node with upstream
			// failure.
		}
	}
	return
}

// Build the specified node (caller has determined that it should be
// built and can be built). On failure, report the error (e.g. to the
// console, a GUI window, ...) and return false. On success, return
// true.
func (self *BuildState) buildNode(node dag.Node, builderr *BuildError) bool {
	rule := node.BuildRule()
	log.Verbose("building node %s, action=%s\n",
		node, rule.ActionString())
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
