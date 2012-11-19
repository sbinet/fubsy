package runtime

import (
	"fmt"
	"strings"
	"fubsy/dsl"
)

type Namespace map[string] FuObject

type Runtime struct {
	script string				// filename
	ast dsl.AST

	locals Namespace
}

type RuntimeError struct {
	location dsl.Location
	message string
}

func NewNamespace() Namespace {
	return make(Namespace)
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	return &Runtime{script: script, ast: ast}
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

func (self *Runtime) evaluate (expr_ dsl.ASTExpression) (FuObject, error) {
	var result FuObject
	var ok bool
	switch expr := expr_.(type) {
	case *dsl.ASTString:
		result = FuString(expr.Value())
	case *dsl.ASTName:
		name := expr.Name()
		result, ok = self.locals[name]
		if !ok {
			err := RuntimeError{
				location: expr.Location(),
				message: fmt.Sprintf("undefined variable '%s'", name),
			}
			return nil, err
		}
	}
	return result, nil
}

func (self *Runtime) addRule(node *dsl.ASTBuildRule) error {
	return nil
}

func (self RuntimeError) Error() string {
	return self.location.String() + self.message
}
