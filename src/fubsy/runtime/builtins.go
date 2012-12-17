// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
	"fmt"
	"os"

	"fubsy/types"
)

// builtin functions (and other objects?)

// Add all builtin functions to ns.
func defineBuiltins(ns types.Namespace) {
	functions := []*types.FuFunction{
		types.NewVariadicFunction("println", 0, -1, fn_println),
		types.NewVariadicFunction("mkdir", 0, -1, fn_mkdir),
		types.NewVariadicFunction("remove", 0, -1, fn_remove),
	}

	for _, function := range functions {
		ns.Assign(function.Name(), function)
	}
}

func fn_println(
	args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, error) {
	for i, val := range args {
		if i > 0 {
			os.Stdout.WriteString(" ")
		}
		_, err := os.Stdout.WriteString(val.String())
		if err != nil {
			return nil, err
		}
	}
	os.Stdout.WriteString("\n")
	return nil, nil
}

func fn_mkdir(
	args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, error) {
	patherrors := make([]error, 0)
	for _, name := range args {
		err := os.MkdirAll(name.String(), 0755)
		if err != nil {
			patherrors = append(patherrors, err)
		}
	}
	if len(patherrors) == 1 {
		return nil, patherrors[0]
	} else if len(patherrors) > 1 {
		message := fmt.Sprintf(
			"error creating %d directories:", len(patherrors))
		for _, err := range patherrors {
			message += "\n  " + err.Error()
		}
		return nil, errors.New(message)
	}
	return nil, nil
}

func fn_remove(
	args []types.FuObject, kwargs map[string]types.FuObject) (
	types.FuObject, error) {
	patherrors := make([]error, 0)
	for _, name := range args {
		err := os.RemoveAll(name.String())
		if err != nil {
			patherrors = append(patherrors, err)
		}
	}
	if len(patherrors) == 1 {
		return nil, patherrors[0]
	} else if len(patherrors) > 1 {
		message := fmt.Sprintf(
			"error creating %d directories:", len(patherrors))
		for _, err := range patherrors {
			message += "\n  " + err.Error()
		}
		return nil, errors.New(message)
	}
	return nil, nil
}
