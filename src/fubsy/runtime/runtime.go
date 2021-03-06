// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

// High-level executive code. This is the home of Runtime, which
// manages everything about the execution of a Fubsy build script (or
// collection of build scripts, once we support hierarchical build
// scripts). There should be exactly one instance of Runtime in one
// Fubsy process.

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"fubsy/build"
	"fubsy/dag"
	"fubsy/db"
	"fubsy/dsl"
	"fubsy/log"
	"fubsy/plugins"
	"fubsy/types"
)

type Runtime struct {
	options build.BuildOptions
	script  string // filename
	ast     *dsl.ASTRoot

	builtins BuiltinList
	stack    *types.ValueStack
	dag      *dag.DAG
}

func NewRuntime(
	options build.BuildOptions, script string, ast *dsl.ASTRoot) *Runtime {
	stack := types.NewValueStack()

	builtins := defineBuiltins()
	stack.Push(builtins)

	// Local variables are per-script, but we only support a single
	// script right now. So might as well initialize the script-local
	// namespace right here.
	locals := types.NewValueMap()
	stack.Push(locals)

	return &Runtime{
		options:  options,
		script:   script,
		ast:      ast,
		builtins: builtins,
		stack:    &stack,
		dag:      dag.NewDAG(),
	}
}

// return a barebones Runtime with almost nothing in it -- variable
// assignment and lookup works, but not much else
func minimalRuntime() *Runtime {
	stack := types.NewValueStack()
	locals := types.NewValueMap()
	stack.Push(locals)
	return &Runtime{
		stack: &stack,
		dag:   dag.NewDAG(),
	}
}

func (self *Runtime) RunScript() []error {
	var errors []error
	for _, plugin := range self.ast.FindImports() {
		log.Debug(log.PLUGINS, "loading plugin '%s'", strings.Join(plugin, "."))
	}

	errors = self.runInlinePlugins()
	if len(errors) > 0 {
		return errors
	}

	errors = self.runMainPhase()
	if len(errors) > 0 {
		return errors
	}

	errors = self.runBuildPhase()
	return errors
}

func (self *Runtime) runInlinePlugins() []error {
	var errs []error
	var err error
	var meta plugins.MetaPlugin

	inlines := self.ast.FindInlinePlugins()
	ns := self.stack.Inner()
	for _, inline := range inlines {
		meta, err = plugins.LoadMetaPlugin(inline.Language(), self.builtins)
		if err != nil {
			errs = append(errs, MakeLocationError(inline, err))
			continue
		}
		log.Debug(log.PLUGINS, "running %s inline plugin", inline.Language())
		values, err := meta.Run(inline.Content())
		if err != nil {
			errs = append(errs, MakeLocationError(inline, err))
		}
		for name, val := range values {
			ns.Assign(name, val)
		}
	}
	return errs
}

// Run all the statements in the main phase of this build script.
// Update self with the results: variable assignments, build rules,
// etc. Most importantly, on return self.dag will contain the
// dependency graph ready to hand over to the build phase.
func (self *Runtime) runMainPhase() []error {
	main := self.ast.FindPhase("main")
	if main == nil {
		return []error{
			MakeLocationError(self.ast.EOF(),
				errors.New("no main phase defined"))}
	}

	var allerrors []error // from the entire phase
	var errs []error      // from a single statement
	for _, node_ := range main.Children() {
		switch node := node_.(type) {
		case *dsl.ASTAssignment:
			errs = self.assign(node)
		case *dsl.ASTBuildRule:
			var rule *BuildRule
			rule, errs = self.makeRule(node)
			if len(errs) == 0 {
				self.addRule(rule)
			}
		case dsl.ASTExpression:
			_, errs = self.evaluate(node)
		default:
			errs = append(errs, unsupportedAST(node))
		}
		errs = MakeLocationErrors(node_, errs)
		allerrors = append(allerrors, errs...)
	}

	return allerrors
}

func (self *Runtime) makeRule(astrule *dsl.ASTBuildRule) (*BuildRule, []error) {
	targets, sources, errs := self.makeRuleNodes(astrule)
	if len(errs) > 0 {
		return nil, errs
	}

	allactions := NewSequenceAction()
	for _, action_ := range astrule.Actions() {
		switch action := action_.(type) {
		case *dsl.ASTString:
			allactions.AddCommand(action)
		case *dsl.ASTAssignment:
			allactions.AddAssignment(action)
		case *dsl.ASTFunctionCall:
			allactions.AddFunctionCall(action)
		}
	}

	rule := NewBuildRule(self, targets, sources)
	rule.action = allactions
	return rule, nil
}

func (self *Runtime) makeRuleNodes(astrule *dsl.ASTBuildRule) (
	targets, sources []dag.Node, errs []error) {

	// Evaluate the target and source lists, so we get one FuObject
	// each. It might be a string, a list of strings, a FinderNode...
	// anything, really.
	var targetobj, sourceobj types.FuObject
	targetobj, errs = self.evaluate(astrule.Targets())
	if len(errs) > 0 {
		return
	}
	sourceobj, errs = self.evaluate(astrule.Sources())
	if len(errs) > 0 {
		return
	}

	// Convert each of those FuObjects to a list of DAG nodes.
	targets = self.nodify(targetobj)
	sources = self.nodify(sourceobj)
	return
}

func (self *Runtime) addRule(rule *BuildRule) {
	targets := rule.targets.Nodes()
	sources := rule.sources.Nodes()

	// Attach the rule to each target node.
	for _, tnode := range targets {
		tnode.SetBuildRule(rule)
	}

	// And connect the nodes to each other (every source is a parent
	// of every target).
	self.dag.AddManyParents(targets, sources)
}

// Convert a single FuObject (possibly a FuList) to a list of Nodes and
// add them to the DAG.
func (self *Runtime) nodify(values types.FuObject) []dag.Node {
	// Blecchh: specially handling every type here limits the
	// extensibility of the type system. But I don't want each type to
	// know how it becomes a node, because then the 'types' package
	// depends on 'dag', which seems backwards to me. Hmmmm.
	var result []dag.Node
	switch values := values.(type) {
	case types.FuString:
		result = []dag.Node{dag.MakeFileNode(self.dag, values.ValueString())}
	case types.FuList:
		result = make([]dag.Node, 0, len(values.List()))
		for _, val := range values.List() {
			result = append(result, self.nodify(val)...)
		}
	case *dag.ListNode:
		result = values.Nodes()
		for i, node := range result {
			result[i] = self.dag.AddNode(node)
		}
	case dag.Node:
		result = []dag.Node{self.dag.AddNode(values)}
	}
	return result
}

// Build user's requested targets according to the dependency graph in
// self.dag (as constructed by runMainPhase()).
func (self *Runtime) runBuildPhase() []error {
	var errs []error

	errs = self.dag.ExpandNodes(self.stack)
	if len(errs) > 0 {
		return errs
	}
	self.dag.MarkSources()

	log.Debug(log.DAG, "dependency graph:")
	log.DebugDump(log.DAG, self.dag)

	goal, errs := self.dag.MatchTargets(self.options.Targets)
	if len(errs) > 0 {
		return errs
	}

	bdb, err := openBuildDB()
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	defer bdb.Close()

	bstate := build.NewBuildState(self.dag, bdb, self.options)
	err = bstate.BuildTargets(goal)
	if err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (self *Runtime) Namespace() types.Namespace {
	return self.stack
}

func (self *Runtime) Lookup(name string) (types.FuObject, bool) {
	return self.stack.Lookup(name)
}

func openBuildDB() (build.BuildDB, error) {
	var bdb build.BuildDB
	var err error

	err = os.MkdirAll(".fubsy", 0755)
	if err != nil {
		return nil, err
	}

	bdb, err = db.OpenKyotoDB(".fubsy/buildstate.kch", true)
	if _, ok := err.(db.NotAvailableError); ok {
		bdb = db.NewFakeDB()
		err = nil
		log.Warning(
			"no database libraries available; build state will not be saved")
	} else if err != nil {
		return nil, err
	}
	return bdb, nil
}

func unsupportedAST(node dsl.ASTNode) error {
	return fmt.Errorf("syntax not supported yet: %v", node)
}
