// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build python

package plugins

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_PythonPlugin_Run(t *testing.T) {
	pp, err := NewPythonPlugin()
	testutils.NoError(t, err)
	values, err := pp.Run(`
import os
foo = ["abc", "def"]
bar = "!".join(foo)
def visible():
    pass
def _hidden():
    pass
pjoin = os.path.join
`)
	testutils.NoError(t, err)

	for _, name := range []string{"os", "foo", "bar", "_hidden", "basdf"} {
		value, ok := values.Lookup(name)
		assert.True(t, value == nil && !ok,
			"expected nothing for name '%s', but got: %v (%T)", name, value, value)
	}
	for _, name := range []string{"visible", "pjoin"} {
		value, ok := values.Lookup(name)
		callable := value.(PythonCallable)
		assert.True(t, ok && callable.Name() == name,
			"expected a PythonCallable for name '%s'", name)
	}

	values, err = pp.Run("foo = 1/0")
	assert.Equal(t, "inline Python plugin raised an exception", err.Error())
}

func Test_PythonPlugin_builtins(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this isn't really the builtin Fubsy println() function: we can't
	// use it because it's in the runtime package, and we don't want
	// to because it has side-effects... but we're stuck with a
	// hardcoded set of builtin function names for now, so we have to
	// reuse one of them
	calls := []string{}
	fn_println := func(args types.ArgSource) (types.FuObject, []error) {
		s := args.Args()[0].ValueString()
		calls = append(calls, s)
		return nil, nil
	}
	builtins := StubBuiltinList{types.NewFixedFunction("println", 1, fn_println)}

	pp, err := LoadMetaPlugin("python2", builtins)
	testutils.NoError(t, err)

	values, err := pp.Run(`
fubsy.println("ding")
fubsy.println("dong")
`)
	_ = values
	expect := []string{"ding", "dong"}
	testutils.NoError(t, err)
	assert.Equal(t, expect, calls)
}

func Test_PythonCallable(t *testing.T) {
	var callable types.FuCallable
	callable = MakePythonCallable("fred", nil)
	assert.Equal(t, "fred", callable.Name())
	assert.Equal(t, "python:fred()", callable.String())
	assert.Equal(t, "fred", callable.ValueString())

	_ = callable.Code()
}

func Test_PythonCallable_callPython(t *testing.T) {
	plugin, err := NewPythonPlugin()
	testutils.NoError(t, err)

	// Setup: define a Python function and make sure that Run() finds
	// it, so it can be added to a Fubsy namespace and used from Fubsy
	// code.
	pycode := `
def reverse_strings(*args):
    '''takes N strings, reverses each one, then returns the reversed
    strings concatenated into a single string with + between each
    one'''
    return '+'.join(''.join(reversed(arg)) for arg in args)`
	values, err := plugin.Run(pycode)
	testutils.NoError(t, err)
	value, ok := values.Lookup("reverse_strings")
	assert.True(t, ok)
	pycallable := value.(PythonCallable)
	assert.Equal(t, "reverse_strings", pycallable.Name())

	// Call the Python function with 3 strings.
	args := types.MakeStringList("foo", "blob", "pingpong").List()
	argsource := types.MakeBasicArgs(nil, args, nil)
	value, errs := pycallable.callPython(argsource)
	testutils.NoErrors(t, errs)

	// And test the returned value.
	expect := types.MakeFuString("oof+bolb+gnopgnip")
	assert.True(t, expect.Equal(value),
		"expected %s, but got %s", expect, value)
}

type StubBuiltinList []types.FuCallable

func (self StubBuiltinList) NumBuiltins() int {
	return len(self)
}

func (self StubBuiltinList) Builtin(idx int) (string, types.FuCode) {
	return self[idx].Name(), self[idx].Code()
}
