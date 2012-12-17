// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

// High-level executive code. This is the home of Runtime, which
// manages everything about the execution of a Fubsy build script (or
// collection of build scripts, once we support hierarchical build
// scripts). There should be exactly one instance of Runtime in one
// Fubsy process.

import (
	"fmt"
	"os"
	"strings"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/types"
)

type Runtime struct {
	script string // filename
	ast    dsl.AST

	globals types.ValueMap
	stack   *types.ValueStack
	dag     *dag.DAG
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	stack := types.NewValueStack()

	// The globals namespace is currently used only for builtins,
	// because the right syntax for assigning globals is not yet
	// decided.
	globals := types.NewValueMap()
	defineBuiltins(globals)
	stack.Push(globals)

	// Local variables are per-script, but we only support a single
	// script right now. So might as well initialize the script-local
	// namespace right here.
	locals := types.NewValueMap()
	stack.Push(locals)

	return &Runtime{
		script:  script,
		ast:     ast,
		globals: globals,
		stack:   &stack,
		dag:     dag.NewDAG(),
	}
}

func (self *Runtime) RunScript() []error {
	var errors []error
	for _, plugin := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", strings.Join(plugin, "."))
	}

	errors = self.runMainPhase()
	if len(errors) > 0 {
		return errors
	}

	errors = self.runBuildPhase()
	return errors
}

// Run all the statements in the main phase of this build script.
// Update self with the results: variable assignments, build rules,
// etc. Most importantly, on return self.dag will contain the
// dependency graph ready to hand over to the build phase.
func (self *Runtime) runMainPhase() []error {
	main := self.ast.FindPhase("main")
	if main == nil {
		return []error{
			RuntimeError{self.ast.Location(), "no main phase defined"}}
	}

	var allerrors []error // from the entire phase
	var errs []error      // from a single statement
	for _, node_ := range main.Children() {
		switch node := node_.(type) {
		case *dsl.ASTAssignment:
			errs = assign(self.stack, node)
		case *dsl.ASTBuildRule:
			var rule *BuildRule
			rule, errs = self.makeRule(node)
			if len(errs) == 0 {
				self.addRule(rule)
			}
		case dsl.ASTExpression:
			_, errs = evaluate(self.stack, node)
		default:
			errs = append(errs, unsupportedAST(node))
		}
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
	// each. It might be a string, a list of strings, a FuFileFinder
	// ... anything, really.
	var targetobj, sourceobj types.FuObject
	targetobj, errs = evaluate(self.stack, astrule.Targets())
	if len(errs) > 0 {
		return
	}
	sourceobj, errs = evaluate(self.stack, astrule.Sources())
	if len(errs) > 0 {
		return
	}

	// Convert each of those FuObjects to a list of DAG nodes.
	targets = self.nodify(targetobj)
	sources = self.nodify(sourceobj)
	return
}

func (self *Runtime) addRule(rule *BuildRule) {

	// Attach the rule to each target node.
	for _, tnode := range rule.targets {
		tnode.SetBuildRule(rule)
	}

	// And connect the nodes to each other (every source is a parent
	// of every target).
	self.dag.AddManyParents(rule.targets, rule.sources)
}

// Convert a single FuObject (possibly a FuList) to a list of Nodes and
// add them to the DAG.
func (self *Runtime) nodify(targets_ types.FuObject) []dag.Node {
	// Blecchh: specially handling every type here limits the
	// extensibility of the type system. But I don't want each type to
	// know how it becomes a node, because then the 'types' package
	// depends on 'dag', which seems backwards to me. Hmmmm.
	var result []dag.Node
	switch targets := targets_.(type) {
	case types.FuString:
		result = []dag.Node{dag.MakeFileNode(self.dag, targets.String())}
	case types.FuList:
		result = make([]dag.Node, 0, len(targets))
		for _, val := range targets {
			result = append(result, self.nodify(val)...)
		}
	case *types.FuFileFinder:
		result = []dag.Node{dag.MakeGlobNode(self.dag, targets)}
	}
	return result
}

// Build user's requested targets according to the dependency graph in
// self.dag (as constructed by runMainPhase()).
func (self *Runtime) runBuildPhase() []error {
	var errors []error

	fmt.Println("\ninitial dag:")
	self.dag.Dump(os.Stdout)

	// eventually we should use the command line to figure out the
	// user's desired targets... but the default will always be to
	// build all final targets, so let's just handle that case for now
	goal := self.dag.FindFinalTargets()
	relevant := self.dag.FindRelevantNodes(goal)

	self.dag, errors = self.dag.Rebuild(relevant, self.stack)
	if len(errors) > 0 {
		return errors
	}
	self.dag.MarkSources()
	fmt.Println("\nrebuilt dag:")
	self.dag.Dump(os.Stdout)

	bstate := self.dag.NewBuildState()
	goal = self.dag.FindFinalTargets()
	err := bstate.BuildTargets(goal)
	if err != nil {
		errors = append(errors, err)
	}
	return errors
}

func unsupportedAST(node dsl.ASTNode) error {
	return RuntimeError{
		location: node.Location(),
		message:  fmt.Sprintf("support not implemented for: %v", node),
	}
}

// XXX this is identical to TypeError in types/basictypes.go:
// factor out a common error type?
type RuntimeError struct {
	location dsl.Location
	message  string
}

func (self RuntimeError) Error() string {
	return self.location.String() + self.message
}
