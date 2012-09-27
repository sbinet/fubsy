package main

import "os"
import "fmt"

import "fubsy"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "not enough arguments")
		os.Exit(2)
	}
	script := os.Args[1]
	ast, err := fubsy.Parse(script)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runtime := fubsy.NewRuntime(script, ast)
	runtime.LoadPlugins()
}
