// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"fubsy/types"
)

// A node that represents the execution of some action, rather than
// its output. Useful because many actions don't have any output, e.g.
// unit tests or static analysis tools. The idea is that you create an
// ActionNode for linting one source file, or running one collection
// of unit tests. The first build executes the appropriate action and
// saves the resulting ActionNode in the database. Then future builds
// don't need to re-run that action until the underlying source files
// change. Result: incremental testing, incremental linting, etc.
type ActionNode struct {
	nodebase
}

func MakeActionNode(dag *DAG, name string) *ActionNode {
	_, node := dag.addNode(NewActionNode(name))
	return node.(*ActionNode)
}

func NewActionNode(name string) *ActionNode {
	return &ActionNode{nodebase: makenodebase(name)}
}

func (self *ActionNode) Typename() string {
	return "ActionNode"
}

func (self *ActionNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*ActionNode)
	return ok && other.name == self.name
}

func (self *ActionNode) Add(other types.FuObject) (types.FuObject, error) {
	return defaultNodeAdd(self, other)
}

func (self *ActionNode) List() []types.FuObject {
	return []types.FuObject{self}
}

func (self *ActionNode) ActionExpand(
	ns types.Namespace, ctx *types.ExpandContext) (
	types.FuObject, error) {
	return defaultNodeActionExpand(self, ns)
}

func (self *ActionNode) copy() Node {
	var c ActionNode = *self
	return &c
}

func (self *ActionNode) Exists() (bool, error) {
	// Must return true in order to force BuildState.BuildTargets() to
	// check if parent nodes have changed.
	return true, nil
}

func (self *ActionNode) Signature() ([]byte, error) {
	// Empty signature because there is no output to hash: however,
	// this means that depending on an ActionNode won't work, since it
	// will appear to never change (hmmmm).
	//
	// Possible alternatives:
	// - combine the signatures of this node's parents... but we don't
	//   have access to them
	// - increment a counter every time this node is built... but that
	//   requires access to the database to get the previous value
	return []byte{}, nil
}
