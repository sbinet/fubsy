// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
	"fmt"
	"os"

	"fubsy/dag"
	"fubsy/types"
)

// builtin functions (and other objects?)

// Add all builtin functions to ns.
func defineBuiltins(ns types.Namespace) {
	functions := []*types.FuFunction{
		types.NewVariadicFunction("println", 0, -1, fn_println),
		types.NewVariadicFunction("mkdir", 0, -1, fn_mkdir),
		types.NewVariadicFunction("remove", 0, -1, fn_remove),

		types.NewFixedFunction("build", 3, fn_build),

		// node factories
		types.NewFixedFunction("FileNode", 1, fn_FileNode),
		types.NewFixedFunction("ActionNode", 1, fn_ActionNode),
	}

	for _, function := range functions {
		ns.Assign(function.Name(), function)
	}
}

func fn_println(args types.ArgSource) (types.FuObject, []error) {
	for i, val := range args.Args() {
		if i > 0 {
			os.Stdout.WriteString(" ")
		}
		var s string
		if val == nil {
			s = "(nil)"
		} else {
			s = val.ValueString()
		}
		_, err := os.Stdout.WriteString(s)
		if err != nil {
			// this shouldn't happen, so bail immediately
			return nil, []error{err}
		}
	}
	os.Stdout.WriteString("\n")
	return nil, nil
}

func fn_mkdir(args types.ArgSource) (types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range args.Args() {
		err := os.MkdirAll(name.ValueString(), 0755)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_remove(args types.ArgSource) (types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range args.Args() {
		err := os.RemoveAll(name.ValueString())
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_build(args types.ArgSource) (types.FuObject, []error) {
	rt := args.(FunctionArgs).runtime
	targets := rt.nodify(args.Arg(0))
	sources := rt.nodify(args.Arg(1))
	actionobj := args.Arg(2)

	fmt.Printf(
		"fn_build():\n"+
			"  targets: %T %v\n"+
			"  sources: %T %v\n"+
			"  actions: %T %v\n",
		targets, targets, sources, sources, actionobj, actionobj)

	var errs []error
	var action Action
	if actionstr, ok := actionobj.(types.FuString); ok {
		action = NewCommandAction(actionstr)
	} else {
		errs = append(errs, errors.New("build(): only command strings supported as actions right now, sorry"))
	}

	rule := NewBuildRule(rt, targets, sources)
	rule.action = action

	return rule, errs
}

func fn_FileNode(args types.ArgSource) (types.FuObject, []error) {
	name := args.Arg(0).ValueString()
	graph := args.(FunctionArgs).Graph()
	return dag.MakeFileNode(graph, name), nil
}

func fn_ActionNode(args types.ArgSource) (types.FuObject, []error) {
	basename := args.Arg(0).ValueString()
	graph := args.(FunctionArgs).Graph()
	return dag.MakeActionNode(graph, basename+":action"), nil
}
