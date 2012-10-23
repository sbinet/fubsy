package main

import "os"
import "fmt"

import "fubsy/dsl"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(2)
	}
	script := os.Args[1]
	ast, err := dsl.Parse(script)
	if ast == nil && err == nil {
		panic("ast == nil && err == nil")
	}
	if err != nil {
	 	fmt.Fprintln(os.Stderr, "parse error:", err)
	 	os.Exit(1)
	}
	if ast != nil {
		fmt.Printf("ast:\n")
		ast.Dump(os.Stdout, "")
	}

/*
	runtime := fubsy.NewRuntime(script, ast)
	runtime.LoadPlugins()
*/
}
