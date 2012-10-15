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
	_ = ast

	infile, err := os.Open(script)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	_ = infile
	result := 0
/*
	tokens, err := fubsy.Scan(script, infile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lexical errors:\n%s\n", err)
		result = 1
	}
	for _, tok := range tokens {
		fmt.Println(tok)
	}

	runtime := fubsy.NewRuntime(script, ast)
	runtime.LoadPlugins()
*/
	os.Exit(result)
}
