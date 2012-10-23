package fubsy

import (
	"fmt"
	"fubsy/dsl"
)

type Runtime struct {
	script string
	ast dsl.AST
}

func NewRuntime(script string, ast dsl.AST) *Runtime {
	return &Runtime{script, ast}
}

func (self Runtime) LoadPlugins() {
	for _, name := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", name)
	}
}
