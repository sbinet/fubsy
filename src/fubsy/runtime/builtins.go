// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
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

		// node constructors
		types.NewFixedFunction("FileNode", 1, fn_FileNode),
		types.NewFixedFunction("ActionNode", 1, fn_ActionNode),
	}

	for _, function := range functions {
		ns.Assign(function.Name(), function)
	}
}

func fn_println(
	robj types.FuObject, args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	for i, val := range args {
		if i > 0 {
			os.Stdout.WriteString(" ")
		}
		_, err := os.Stdout.WriteString(val.String())
		if err != nil {
			// this shouldn't happen, so bail immediately
			return nil, []error{err}
		}
	}
	os.Stdout.WriteString("\n")
	return nil, nil
}

func fn_mkdir(
	robj types.FuObject, args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range args {
		err := os.MkdirAll(name.String(), 0755)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_remove(
	robj types.FuObject, args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range args {
		err := os.RemoveAll(name.String())
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_FileNode(
	robj types.FuObject, args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	return dag.NewFileNode(args[0].String()), nil
}

func fn_ActionNode(
	robj types.FuObject, args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	basename := args[0].String()
	return dag.NewActionNode(basename + ":action"), nil
}
