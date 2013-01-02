// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"fmt"
	"os"

	"fubsy/dag"
	"fubsy/types"
)

type BuildRule struct {
	runtime *Runtime
	targets []dag.Node
	sources []dag.Node
	action  Action
	locals  types.ValueMap
}

func NewBuildRule(runtime *Runtime, targets, sources []dag.Node) *BuildRule {
	return &BuildRule{
		runtime: runtime,
		targets: targets,
		sources: sources,
	}
}

func (self *BuildRule) Execute() ([]dag.Node, []error) {
	stack := self.runtime.stack
	locals := types.NewValueMap()
	stack.Push(locals)
	defer stack.Pop()

	self.setLocals(locals)
	fmt.Printf("about to execute action %v; stack:\n", self.action)
	stack.Dump(os.Stdout, "")
	err := self.action.Execute(stack)
	return self.targets, err
}

func (self *BuildRule) ActionString() string {
	return self.action.String()
}

func (self *BuildRule) setLocals(ns types.Namespace) {
	// Convert each slice-of-nodes to a FuList
	targets := make(types.FuList, len(self.targets))
	for i, tnode := range self.targets {
		targets[i] = tnode
	}
	sources := make(types.FuList, len(self.sources))
	for i, snode := range self.sources {
		sources[i] = snode
	}

	ns.Assign("TARGETS", targets)
	ns.Assign("SOURCES", sources)

	// these are really only meaningful for rules with one target or
	// one source... but such rules are pretty common, so these are
	// frequently handy
	ns.Assign("TARGET", targets[0])
	ns.Assign("SOURCE", sources[0])
}
