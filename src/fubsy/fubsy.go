// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ogier/pflag"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/runtime"
)

type args struct {
	options    dag.BuildOptions
	scriptFile string
	targets    []string
}

func main() {
	args := parseArgs()
	script, err := findScript(args.scriptFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fubsy: error: "+err.Error())
		os.Exit(2)
	}

	ast, errors := dsl.Parse(script)
	if ast == nil && len(errors) == 0 {
		panic("ast == nil && len(errors) == 0")
	}
	checkErrors("parse error:", errors)
	fmt.Printf("ast:\n")
	ast.Dump(os.Stdout, "")

	rt := runtime.NewRuntime(args.options, script, ast)
	errors = rt.RunScript()
	checkErrors("error:", errors)
}

func usage() {
	fmt.Printf("Usage: %s [options] [target ...]\n", filepath.Base(os.Args[0]))
	fmt.Println(`
Build out-of-date targets from sources by executing actions defined in
a build script according to the dependencies between sources and
targets.

Options:
  -k, --keep-going         continue building even when some targets fail
  --check-all              check all files for changes, not just sources
  -f FILE, --file=FILE     read build script from FILE (default: main.fubsy)
`)
}

func parseArgs() args {
	result := args{}
	pflag.Usage = usage
	pflag.BoolVarP(&result.options.KeepGoing, "keep-going", "k", false, "")
	pflag.BoolVar(&result.options.CheckAll, "check-all", false, "")
	pflag.StringVarP(&result.scriptFile, "file", "f", "", "")
	pflag.Parse()
	result.targets = pflag.Args()
	return result
}

func findScript(script string) (string, error) {
	if script != "" {
		// user specified the script on the command line
		return script, nil
	} else if isFile("main.fubsy") {
		// default script name: this is Fubsy's equivalent to
		// "Makefile" or "SConstruct" or "build.xml"
		return "main.fubsy", nil
	}

	// allow fallback to arbitrary *.fubsy name, as long as there is
	// exactly one
	matches, err := filepath.Glob("*.fubsy")
	if err != nil {
		return "", err
	}
	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		return "", errors.New(
			"main.fubsy not found, and multiple *.fubsy files exist " +
				"(use -f to pick one)")
	}
	return "", errors.New(
		"main.fubsy not found (and no other *.fubsy files found)")
}

func isFile(name string) bool {
	fileinfo, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !fileinfo.IsDir()
}

func checkErrors(prefix string, errors []error) {
	if len(errors) > 0 {
		for _, err := range errors {
			fmt.Fprintln(os.Stderr, prefix, err)
		}
		os.Exit(1)
	}
}
