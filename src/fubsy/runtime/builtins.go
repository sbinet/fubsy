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
		types.NewFixedFunction("ActionNode", 1, fn_ActionNode),
	}

	for _, function := range functions {
		ns.Assign(function.Name(), function)
	}
}

func fn_println(
	args []types.FuObject, kwargs map[string]types.FuObject) (
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
	args []types.FuObject, kwargs map[string]types.FuObject) (
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
	args []types.FuObject, kwargs map[string]types.FuObject) (
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

func fn_ActionNode(
	args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, []error) {
	return dag.NewActionNode(args[0].String()), nil
}
