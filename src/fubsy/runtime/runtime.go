package runtime

import (
	"fmt"
	"os"
	"strings"
	"fubsy/dsl"
	"fubsy/types"
	"fubsy/dag"
)

type Namespace map[string] types.FuObject

type Runtime struct {
	script string				// filename
	ast dsl.AST

	locals Namespace
	dag *dag.DAG
}

func NewNamespace() Namespace {
	return make(Namespace)
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	return &Runtime{
		script: script,
		ast: ast,
		locals: NewNamespace(),
		dag: dag.NewDAG(),
	}
}

func (self *Runtime) RunScript() []error {
	for _, plugin := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", strings.Join(plugin, "."))
	}
	main := self.ast.FindPhase("main")
	errors := self.runStatements(main)
	if len(errors) > 0 {
		return errors
	}

	errors = self.buildTargets()
	return errors
}

// Run all the statements in the main phase of this build script.
// Update self with the results: variable assignments, build rules,
// etc. Most importantly, on return self.dag will contain the
// dependency graph ready to hand over to the build phase.
func (self *Runtime) runStatements(main *dsl.ASTPhase) []error {
	errors := make([]error, 0)
	for _, node_ := range main.Children() {
		var err error
		switch node := node_.(type) {
		case *dsl.ASTAssignment:
			err = self.assign(node, self.locals)
		case *dsl.ASTBuildRule:
			rule, err := self.makeRule(node)
			if err == nil {
				self.addRule(rule)
			}
		}

		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

// node represents code like "NAME = EXPR": evaluate EXPR and store
// the result in ns
func (self *Runtime) assign(node *dsl.ASTAssignment, ns Namespace) error {
	value, err := self.evaluate(node.Expression())
	if err != nil {
		return err
	}
	ns[node.Target()] = value
	return nil
}

func (self *Runtime) evaluate (expr_ dsl.ASTExpression) (types.FuObject, error) {
	switch expr := expr_.(type) {
	case *dsl.ASTString:
		return types.FuString(expr.Value()), nil
	case *dsl.ASTName:
		return self.evaluateName(expr)
	case *dsl.ASTFileList:
		return types.NewFileFinder(expr.Patterns()), nil
	case *dsl.ASTAdd:
		return self.evaluateAdd(expr)
	default:
		panic(fmt.Sprintf("unknown expression type: %T", expr))
	}
	panic("unreachable code")
}

func (self *Runtime) evaluateName(expr *dsl.ASTName) (types.FuObject, error) {
	name := expr.Name()
	if result, ok := self.locals[name]; ok {
		return result, nil
	}
	err := RuntimeError{
		location: expr.Location(),
		message: fmt.Sprintf("undefined variable '%s'", name),
	}
	return nil, err
}

func (self *Runtime) evaluateAdd(expr *dsl.ASTAdd) (types.FuObject, error) {
	op1, op2 := expr.Operands()
	obj1, err := self.evaluate(op1)
	if err != nil {
		return nil, err
	}
	obj2, err := self.evaluate(op2)
	if err != nil {
		return nil, err
	}
	return obj1.Add(obj2)
}

func (self *Runtime) makeRule(node *dsl.ASTBuildRule) (*BuildRule, error) {
	// Evaluate the target and source lists, so we get one FuObject
	// each. It might be a string, a list of strings, or a
	// FuFileFinder... we just need to be able to get one or more
	// filenames out of each.
	fmt.Printf("adding build rule\n")
	targets, err := self.evaluate(node.Targets())
	if err != nil {
		return nil, err
	}
	fmt.Printf("targets = %T %v, err = %v\n", targets, targets, err)
	sources, err := self.evaluate(node.Sources())
	if err != nil {
		return nil, err
	}
	fmt.Printf("sources = %T %v, err = %v\n", sources, sources, err)

	allactions := dag.NewSequenceAction()
	for _, action_ := range node.Actions() {
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

func (self *Runtime) addRule(rule *BuildRule) error {
	// Convert a single FuObject representing the targets (sources) --
	// could be a single string (filename), list of strings,
	// FuFileFinder, or FuFinderList -- to a list of Nodes in the DAG.
	targets := self.nodify(rule.targets)
	sources := self.nodify(rule.sources)

	// Attach the rule's action to each target node.
	for _, tnode := range targets {
		tnode.SetAction(rule.action)
	}

	// And connect the nodes to each other (every source is a parent
	// of every target).
	self.dag.AddManyParents(targets, sources)

	// umm: when can this fail?
	return nil
/*
	// XXX this violates the definition of Expand(): it's supposed to
	// happen in the build phase, but this code runs in the main phase.
	targets, err := rule.targets.Expand(self)
	if err != nil {
		return err
	}
	sources, err := rule.sources.Expand(self)
	if err != nil {
		return err
	}

	// XXX assuming all sources and targets are regular files -- what
	// if they are directories? symlinks? don't exist anymore (deleted
	// in the window between Expand() and now)?

	snodes := make([]dag.Node, len(sources))
	for i, source := range sources {
		snodes[i] = dag.MakeFileNode(self.dag)
	}
	tnodes := make([]dag.Node, len(targets))
	for i, target := range targets {
		tnodes[i] = dag.MakeFileNode(self.dag, target)
		tnodes[i].SetAction(action)
	}

	self.dag.AddManyParents(tnodes, snodes)
*/
}

// Convert a single FuObject (possibly a FuList or FuFinderList) to a
// list of Nodes in the DAG.
func (self *Runtime) nodify(targets_ types.FuObject) []dag.Node {
	// Blecchh: this limits the extensibility of the type system if we
	// have handle every type specially here. But I don't want each
	// type to know how it becomes a node, because then the 'types'
	// package depends on 'dag', which seems backwards to me. Hmmmm.
	var result []dag.Node
	switch targets := targets_.(type) {
	case types.FuString:
		result = []dag.Node {dag.MakeFileNode(self.dag, targets.Value())}
	case types.FuList:
		filenames := targets.Values()
		result := make([]dag.Node, len(filenames))
		for i, fn := range filenames {
			result[i] = dag.MakeFileNode(self.dag, fn)
		}
	case *types.FuFileFinder:
	case *types.FuFinderList:
		result = []dag.Node {dag.MakeGlobNode(self.dag, targets)}
	}
	return result
}

// Build user's requested targets according to the dependency graph in
// self.dag (as constructed by runStatements()).
func (self *Runtime) buildTargets() []error {
	var errors []error

	fmt.Println("\ninitial dag:")
	self.dag.Dump(os.Stdout)

	// eventually we should use the command line to figure out the
	// user's desired targets... but the default will always be to
	// build all final targets, so let's just handle that case for now
	goal := self.dag.FindFinalTargets()
	relevant := dag.FindRelevantNodes(self.dag, goal)

	self.dag, errors = self.dag.Rebuild(relevant)
	if len(errors) > 0 {
		return errors
	}
	fmt.Println("\nrebuilt dag:")
	self.dag.Dump(os.Stdout)

	// Now that we've expanded the DAG, rediscover targets, sources,
	// etc.
	self.dag.ComputeChildren()
	bstate := self.dag.NewBuildState()
	// goal = self.dag.FindFinalTargets()
	// fmt.Printf("goal (v2) = %v\n", goal)
	// relevant = dag.FindRelevantNodes(self.dag, goal)

	stale, errors := dag.FindStaleTargets(self.dag)
	if len(errors) > 0 {
		return errors
	}
	err := bstate.BuildStaleTargets(stale)
	if err != nil {
		return []error {err}
	}
	return nil
}

// XXX this is identical to TypeError in types/basictypes.go:
// factor out a common error type?
type RuntimeError struct {
	location dsl.Location
	message string
}

func (self RuntimeError) Error() string {
	return self.location.String() + self.message
}
