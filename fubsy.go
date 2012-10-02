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

	infile, err := os.Open(script)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	tokens, err := fubsy.Scan(script, infile)
	for _, tok := range tokens {
		fmt.Println(tok)
	}

	runtime := fubsy.NewRuntime(script, ast)
	runtime.LoadPlugins()
}
