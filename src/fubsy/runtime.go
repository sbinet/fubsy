package fubsy

import "fmt"

type Runtime struct {
	script string
	ast AST
}

func NewRuntime(script string, ast AST) *Runtime {
	return &Runtime{script, ast}
}

func (self Runtime) LoadPlugins() {
	for _, name := range self.ast.ListPlugins() {
		fmt.Printf("loading plugin %s\n", name)
	}
}
