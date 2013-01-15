// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ogier/pflag"

	"fubsy/build"
	"fubsy/dsl"
	"fubsy/log"
	"fubsy/runtime"
)

type args struct {
	options     build.BuildOptions
	scriptFile  string
	debugTopics []string
	verbosity   uint
}

func main() {
	if filepath.Base(os.Args[0]) == "fubsydebug" {
		debugmain()
		return
	}

	args := parseArgs()
	script, err := findScript(args.scriptFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fubsy: error: "+err.Error())
		os.Exit(2)
	}
	log.SetVerbosity(args.verbosity)
	err = log.EnableDebugTopics(args.debugTopics)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fubsy: error: "+err.Error())
		os.Exit(2)
	}

	ast, errors := dsl.Parse(script)
	if ast == nil && len(errors) == 0 {
		panic("ast == nil && len(errors) == 0")
	}
	checkErrors("parse error:", errors)
	log.Debug(log.AST, "ast:\n")
	log.DebugDump(log.AST, ast)

	rt := runtime.NewRuntime(args.options, script, ast)
	errors = rt.RunScript()
	checkErrors("error:", errors)
}

func usage() {
	fmt.Printf("Usage: %s [options] [target ...]\n", filepath.Base(os.Args[0]))
	topics := strings.Join(log.TopicNames(), ", ")
	help := `
Build out-of-date targets from sources by executing actions defined in
a build script according to the dependencies between sources and
targets.

Options:
  -k, --keep-going         continue building even when some targets fail
  --check-all              check all files for changes, not just sources
  -f FILE, --file=FILE     read build script from FILE (default: main.fubsy)
  -v, --verbose            print more informative messages
  -q, --quiet              suppress all non-error output
  --debug=TOPIC,...        print detailed debug info about TOPIC: one of
                           ` + topics + `
                           (specify multiple topics as a comma-separated list)`

	fmt.Println(help)
}

func parseArgs() args {
	result := args{}
	pflag.Usage = usage
	pflag.BoolVarP(&result.options.KeepGoing, "keep-going", "k", false, "")
	pflag.BoolVar(&result.options.CheckAll, "check-all", false, "")
	pflag.StringVarP(&result.scriptFile, "file", "f", "", "")
	verbose := pflag.BoolP("verbose", "v", false, "")
	quiet := pflag.BoolP("quiet", "q", false, "")
	topics := pflag.String("debug", "", "")
	pflag.Parse()
	if *topics != "" {
		result.debugTopics = strings.Split(*topics, ",")
	}

	// argh: really, we just want a callback for each occurence of -q
	// or -v, which decrements or increments verbosity
	if *quiet {
		result.verbosity = 0
	} else if *verbose {
		result.verbosity = 2
	} else {
		result.verbosity = 1
	}

	result.options.Targets = pflag.Args()
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
