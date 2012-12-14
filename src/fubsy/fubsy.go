// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"os"

	"fubsy/dsl"
	"fubsy/runtime"
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
	checkErrors("parse error:", errors)
	fmt.Printf("ast:\n")
	ast.Dump(os.Stdout, "")

	rt := runtime.NewRuntime(script, ast)
	errors = rt.RunScript()
	checkErrors("error:", errors)
}

func checkErrors(prefix string, errors []error) {
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, prefix, err)
		}
		os.Exit(1)
	}
}
