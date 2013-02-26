// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
	"fmt"
	"io"
	"os"

	"fubsy/dag"
	"fubsy/types"
)

// builtin functions (and other objects?)

// implements types.Namespace, plugins.BuiltinList
type BuiltinList struct {
	builtins []types.FuCallable
}

// Namespace methods
func (self BuiltinList) Lookup(name string) (types.FuObject, bool) {
	idx, callable := self.find(name)
	if idx == -1 {
		return nil, false
	}
	return callable, true
}

func (self BuiltinList) Assign(name string, value types.FuObject) {
	panic("list of builtin functions is immutable")
}

func (self BuiltinList) ForEach(
	callback func(name string, value types.FuObject) error) error {
	for _, b := range self.builtins {
		err := callback(b.Name(), b)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self BuiltinList) Dump(writer io.Writer, indent string) {
	for _, b := range self.builtins {
		fmt.Fprintf(writer, "%s%s = %s\n", indent, b.Name(), b)
	}
}

// private methods
func (self BuiltinList) find(name string) (int, types.FuCallable) {
	for i, b := range self.builtins {
		if b.Name() == name {
			return i, b
		}
	}
	return -1, nil
}

// methods of plugins.BuiltinList
func (self BuiltinList) NumBuiltins() int {
	return len(self.builtins)
}

func (self BuiltinList) Builtin(idx int) (string, types.FuCode) {
	// panic if idx out of range
	b := self.builtins[idx]
	return b.Name(), b.Code()
}

// Return a new namespace containing all builtin functions.
func defineBuiltins() BuiltinList {
	builtins := []types.FuCallable{
		types.NewVariadicFunction("println", 0, -1, fn_println),
		types.NewVariadicFunction("mkdir", 0, -1, fn_mkdir),
		types.NewVariadicFunction("remove", 0, -1, fn_remove),

		types.NewFixedFunction("build", 3, fn_build),

		// node factories
		types.NewFixedFunction("FileNode", 1, fn_FileNode),
		types.NewFixedFunction("ActionNode", 1, fn_ActionNode),
	}
	return BuiltinList{builtins}
}

func fn_println(argsource types.ArgSource) (types.FuObject, []error) {
	for i, val := range argsource.Args() {
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

func fn_mkdir(argsource types.ArgSource) (types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range argsource.Args() {
		err := os.MkdirAll(name.ValueString(), 0755)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_remove(argsource types.ArgSource) (types.FuObject, []error) {
	errs := make([]error, 0)
	for _, name := range argsource.Args() {
		err := os.RemoveAll(name.ValueString())
		if err != nil {
			errs = append(errs, err)
		}
	}
	return nil, errs
}

func fn_build(argsource types.ArgSource) (types.FuObject, []error) {
	rt := argsource.(RuntimeArgs).runtime
	args := argsource.Args()
	targets := rt.nodify(args[0])
	sources := rt.nodify(args[1])
	actionobj := args[2]

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

func fn_FileNode(argsource types.ArgSource) (types.FuObject, []error) {
	name := argsource.Args()[0].ValueString()
	graph := argsource.(RuntimeArgs).Graph()
	return dag.MakeFileNode(graph, name), nil
}

func fn_ActionNode(argsource types.ArgSource) (types.FuObject, []error) {
	basename := argsource.Args()[0].ValueString()
	graph := argsource.(RuntimeArgs).Graph()
	return dag.MakeActionNode(graph, basename+":action"), nil
}
