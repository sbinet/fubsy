package runtime

import (
	"fmt"
	"strings"
	"fubsy/dsl"
)

type Runtime struct {
	script string				// filename
	ast dsl.AST

	locals map[string] FuObject
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
			err = self.assign(node)
		case *dsl.ASTBuildRule:
			err = self.addRule(node)
		}

		if err != nil {
			errors = append(errors, err)
		}
	}
	return errors
}

func (self *Runtime) assign(node *dsl.ASTAssignment) error {
	value, err := self.evaluate(node.Expression())
	if err != nil {
		return err
	}
	self.locals[node.Target()] = value
	return nil
}

func (self *Runtime) evaluate (expr dsl.ASTExpression) (FuObject, error) {
	return nil, nil
}

func (self *Runtime) addRule(node *dsl.ASTBuildRule) error {
	return nil
}
