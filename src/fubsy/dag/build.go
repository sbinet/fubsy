// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fmt"
	"os"
	"strings"

	"code.google.com/p/go-bit/bit"

	"fubsy/types"
)

type BuildState struct {
	dag *DAG

	// not much else here now, but at some point we're going to need a
	// place to put user options -- how do we transfer "--keep-going"
	// from the command line to keepGoing() below?
}

type BuildError struct {
	// nodes that failed to build
	failed []Node

	// total number of nodes that we attempted to build
	attempts int
}

// The heart of Fubsy: do a depth-first walk of the dependency graph
// to discover nodes in topological order, then (re)build nodes that
// are stale or missing. Skip target nodes that are "tainted" by
// upstream failure. Returns a single error object summarizing what
// (if anything) went wrong; error details are reported "live" as
// builds fail (e.g. to the console or a GUI window) so the user gets
// timely feedback.
func (self *BuildState) BuildTargets(stack types.NamespaceStack, targets NodeSet) error {
	// What sort of nodes do we check for changes?
	changestates := self.getChangeStates()

	// Need an additional namespace for magic variables ($TARGET, $SOURCES, ...)
	locals := types.NewValueMap()
	stack.Push(locals)
	defer stack.Pop()

	builderr := new(BuildError)
	visit := func(id int) error {
		node := self.dag.nodes[id]
		if node.State() == SOURCE {
			// can't build original source nodes!
			return nil
		}

		checkInitialState(node)

		// do we need to build this node? can we?
		missing, stale, tainted, err :=
			self.inspectParents(changestates, id, node)
		if err != nil {
			return err
		}

		if tainted {
			node.SetState(TAINTED)
		} else if missing || stale {
			self.setLocals(locals, id, node)
			ok := self.buildNode(stack, id, node, builderr)
			if !ok && !self.keepGoing() {
				// attempts counter is not very useful when we break
				// out of the build early
				builderr.attempts = -1
				return builderr
			}
		}
		return nil
	}
	err := self.dag.DFS(targets, visit)
	if err == nil && len(builderr.failed) > 0 {
		// build failures in keep-going mode
		err = builderr
	}
	return err
}

func (self *BuildState) getChangeStates() map[NodeState]bool {
	// Default is like tup: only check original source nodes and nodes
	// that have just been built.
	changestates := make(map[NodeState]bool)
	changestates[SOURCE] = true
	changestates[BUILT] = true
	if self.checkAll() {
		// Optional SCons-like behaviour: assume the user has been
		// sneaking around and modifying .o or .class files behind our
		// back and check everything. (N.B. this is unnecessary if the
		// user sneakily *removes* intermediate targets; that case
		// should be handled just fine by default.)
		changestates[UNKNOWN] = true
	}
	return changestates
}

// panic if node is in an impossible state for starting its visit
func checkInitialState(node Node) {
	if node.State() != UNKNOWN {
		// we just skipped SOURCE nodes, the other states are only set
		// while visiting a node, and we should only visit each node
		// once
		panic(fmt.Sprintf(
			"visiting node %v, state = %d (should be UNKNOWN = %d)",
			node, node.State(), UNKNOWN))
	}
}

// Inspect node and its parents to see if we need to build it. Return
// tainted=true if we should skip building this node due to upstream
// failure. Return stale=true if we should build this node because at
// least one of its parents has changed. Return missing=true if we
// should build this node because its resource is missing. Return
// non-nil err if there were unexpected node errors (error checking
// existence or change status).
func (self *BuildState) inspectParents(
	changestates map[NodeState]bool, id int, node Node) (
	missing, stale, tainted bool, err error) {

	var exists, changed bool
	exists, err = node.Exists() // obvious rebuild (unless tainted)
	if err != nil {
		return
	}
	missing = !exists
	stale = false   // need to rebuild this node
	tainted = false // failures upstream: do not rebuild

	parentnodes := self.dag.parentNodes(id)
	for _, parent := range parentnodes {
		pstate := parent.State()
		if pstate == FAILED || pstate == TAINTED {
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

func (self *BuildState) setLocals(ns types.Namespace, id int, node Node) {
	// XXX how do we set $TARGETS? that requires knowing all nodes
	// that will be built from the same Action as this node!

	// collapsing Nodes to FuStrings loses information: would be nice
	// if every Node type also implemented FuObject!
	pnodes := self.dag.parentNodes(id)
	parents := make([]types.FuObject, len(pnodes))
	for i, pnode := range pnodes {
		parents[i] = types.FuString(pnode.String())
	}

	ns.Assign("TARGET", types.FuString(node.String()))
	ns.Assign("SOURCES", types.FuList(parents))
}

// Build the specified node (caller has determined that it should be
// built and can be built). On failure, report the error (e.g. to the
// console, a GUI window, ...) and return false. On success, return
// true.
func (self *BuildState) buildNode(
	ns types.Namespace, id int, node Node, builderr *BuildError) bool {
	node.SetState(BUILDING)
	builderr.attempts++
	err := node.Action().Execute(ns)
	if err != nil {
		// Normal, everyday build failure: report the precise problem
		// immediately, and accumulate summary info in the caller.
		node.SetState(FAILED)
		self.reportFailure(err)
		builderr.addFailure(node)
		return false
	}
	node.SetState(BUILT)
	return true
}

func (self *BuildState) reportFailure(err error) {
	fmt.Fprintf(os.Stderr, "build failure: %s\n", err)
}

func (self *BuildState) keepGoing() bool {
	// eventually this should come from command-line option -k
	return true
}

func (self *BuildState) checkAll() bool {
	// similarly, this should come from --check-all option
	// (pessimistically assume that someone might have modified
	// intermediate targets behind our back, not just original
	// sources)
	return false
}

func (self *BuildError) addFailure(node Node) {
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

// (hopefully) temporary, pending acceptance of my patches to go-bit
func setToSlice(set *bit.Set) []int {
	result := make([]int, set.Size())
	j := 0
	set.Do(func(n int) {
		result[j] = n
		j++
	})
	return result
}

func joinNodes(delim string, max int, nodes []Node) string {
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
