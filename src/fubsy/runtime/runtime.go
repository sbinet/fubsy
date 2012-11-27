package runtime

import (
	"fmt"
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
}

func NewNamespace() Namespace {
	return make(Namespace)
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	return &Runtime{
		script: script,
		ast: ast,
		locals: NewNamespace(),
	}
}

func (self *Runtime) RunScript() []error {
	for _, plugin := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", strings.Join(plugin, "."))
	}
	main := self.ast.FindPhase("main")
	errors := self.runStatements(main)

	return errors
}

func (self *Runtime) runStatements(main *dsl.ASTPhase) []error {
	errors := make([]error, 0)
	for _, node_ := range main.Children() {
		var err error
		switch node := node_.(type) {
		case *dsl.ASTAssignment:
			err = self.assign(node, self.locals)
		case *dsl.ASTBuildRule:
			err = self.addRule(node)
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

func (self *Runtime) addRule(node *dsl.ASTBuildRule) error {
	// Evaluate the target and source lists, so we get one FuObject
	// each. It might be a string, a list of strings, or a
	// FuFileFinder... we just need to be able to get one or more
	// filenames out of each.
	fmt.Printf("adding build rule\n")
	targets, err := self.evaluate(node.Targets())
	if err != nil {
		return err
	}
	fmt.Printf("targets = %T %v, err = %v\n", targets, targets, err)
	sources, err := self.evaluate(node.Sources())
	if err != nil {
		return err
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
