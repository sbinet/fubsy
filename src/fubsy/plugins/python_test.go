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
	assert.Nil(t, err)
	values, err := pp.Run(`
foo = ["abc", "def"]
bar = "!".join(foo)`)

	// PythonPlugin doesn't yet harvest Python values, so we cannot do
	// anything to test values
	_ = values
	assert.Nil(t, err)

	values, err = pp.Run("foo = 1/0")
	assert.Equal(t, "inline Python plugin raised an exception", err.Error())
}

func Test_PythonPlugin_builtins(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// this isn't really the builtin Fubsy mkdir() function: we can't
	// use it because it's in the runtime package, and we don't want
	// to because it has side-effects... but we're stuck with a
	// hardcoded set of builtin function names for now, so we have to
	// reuse one of them
	calls := []string{}
	fn_mkdir := func(args types.ArgSource) (types.FuObject, []error) {
		s := args.Args()[0].ValueString()
		calls = append(calls, s)
		return nil, nil
	}
	builtins := types.NewValueMap()
	builtins.Assign("mkdir", types.NewFixedFunction("mkdir", 1, fn_mkdir))

	pp, err := LoadMetaPlugin("python2", builtins)
	assert.Nil(t, err)

	values, err := pp.Run(`
fubsy.mkdir("ding")
fubsy.mkdir("dong")
`)
	_ = values
	expect := []string{"ding", "dong"}
	assert.Nil(t, err)
	assert.Equal(t, expect, calls)
}
