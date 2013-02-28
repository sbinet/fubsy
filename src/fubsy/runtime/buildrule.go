// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
	"fmt"

	"fubsy/dag"
	"fubsy/log"
	"fubsy/types"
)

type BuildRule struct {
	runtime *Runtime
	targets *dag.ListNode
	sources *dag.ListNode
	action  Action
	locals  types.ValueMap
	attrs   types.ValueMap
}

func NewBuildRule(runtime *Runtime, targets, sources []dag.Node) *BuildRule {
	rule := &BuildRule{
		runtime: runtime,
		targets: dag.ListNodeFromNodes(targets),
		sources: dag.ListNodeFromNodes(sources),
	}
	rule.attrs = types.NewValueMap()
	rule.attrs["targets"] = rule.targets
	rule.attrs["sources"] = rule.sources
	return rule
}

func (self *BuildRule) Execute() ([]dag.Node, []error) {
	stack := self.runtime.stack
	locals := types.NewValueMap()
	stack.Push(locals)
	defer stack.Pop()

	self.setLocals(locals)
	log.Debug(log.BUILD, "value stack:")
	log.DebugDump(log.BUILD, stack)
	err := self.action.Execute(self.runtime)
	return self.targets.Nodes(), err
}

func (self *BuildRule) ActionString() string {
	return self.action.String()
}

func (self *BuildRule) setLocals(ns types.Namespace) {
	ns.Assign("TARGETS", self.targets)
	ns.Assign("SOURCES", self.sources)

	// these are really only meaningful for rules with one target or
	// one source... but such rules are pretty common, so these are
	// frequently handy
	ns.Assign("TARGET", self.targets.Nodes()[0])
	ns.Assign("SOURCE", self.sources.Nodes()[0])
}

// Implement FuObject so we can expose BuildRules to the DSL
func (self *BuildRule) Typename() string {
	return "BuildRule"
}

func (self *BuildRule) String() string {
	return fmt.Sprintf("%v: %v {%v}", self.targets, self.sources, self.action)
}

func (self *BuildRule) ValueString() string {
	return self.String()
}

func (self *BuildRule) CommandString() string {
	return self.String()
}

func (self *BuildRule) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*BuildRule)
	return ok && self == other
}

func (self *BuildRule) Add(other types.FuObject) (types.FuObject, error) {
	return nil, errors.New("BuildRule objects cannot be added")
}

func (self *BuildRule) Lookup(name string) (types.FuObject, bool) {
	value, ok := self.attrs[name]
	return value, ok
}

func (self *BuildRule) List() []types.FuObject {
	return []types.FuObject{self}
}

func (self *BuildRule) ActionExpand(
	ns types.Namespace, ctx *types.ExpandContext) (
	types.FuObject, error) {
	return nil, errors.New("BuildRule objects cannot be expanded")
}
