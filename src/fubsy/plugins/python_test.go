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
foo = ["abc", "def"]
bar = "!".join(foo)`)

	// PythonPlugin doesn't yet harvest Python values, so we cannot do
	// anything to test values
	_ = values
	testutils.NoError(t, err)

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

type StubBuiltinList []types.FuCallable

func (self StubBuiltinList) NumBuiltins() int {
	return len(self)
}

func (self StubBuiltinList) Builtin(idx int) (string, types.FuCode) {
	return self[idx].Name(), self[idx].Code()
}
