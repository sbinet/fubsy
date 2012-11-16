package main

import (
	"os"
	"fmt"

	"fubsy"
	"fubsy/dsl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(2)
	}
	script := os.Args[1]
	ast, errors := dsl.Parse(script)
	if ast == nil && len(errors) == 0 {
		panic("ast == nil && len(errors) == 0")
	}
	if len(errors) > 0 {
		for _, err := range errors {
	 		fmt.Fprintln(os.Stderr, "parse error:", err)
		}
	 	os.Exit(1)
	}
	fmt.Printf("ast:\n")
	ast.Dump(os.Stdout, "")

	runtime := fubsy.NewRuntime(script, ast)
	runtime.RunScript()
}
