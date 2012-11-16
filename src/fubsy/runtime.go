package fubsy

import (
	"fmt"
	"strings"
	"fubsy/dsl"
)

type Runtime struct {
	script string
	ast dsl.AST
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	return &Runtime{script, ast}
}

func (self *Runtime) RunScript() error {
	for _, plugin := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", strings.Join(plugin, "."))
	}
	main := self.ast.FindPhase("main")
	_ = main

	return nil
}
