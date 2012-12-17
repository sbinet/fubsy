// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_FuFunction_constructors(t *testing.T) {
	fn := NewFixedFunction("foo", 37, nil)
	assert.Equal(t, "foo", fn.name)
	assert.Equal(t, "foo()", fn.String())
	assert.Equal(t, 37, fn.minargs)
	assert.Equal(t, 37, fn.maxargs)

	fn = NewVariadicFunction("bar", 0, 3, nil)
	assert.Equal(t, 0, fn.minargs)
	assert.Equal(t, 3, fn.maxargs)
}

func Test_FuFunction_CheckArgs_fixed(t *testing.T) {
	val := FuString("a")
	args := []FuObject{}
	fn := NewFixedFunction("meep", 0, nil)

	err := fn.CheckArgs(args)
	assert.Nil(t, err)

	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function meep() takes no arguments (got 1)", err.Error())

	fn = NewFixedFunction("foo", 2, nil)
	args = args[:0]
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 0)", err.Error())

	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 1)", err.Error())

	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 3)", err.Error())
}

func Test_FuFunction_CheckArgs_minmax(t *testing.T) {
	fn := NewVariadicFunction("bar", 2, 4, nil)
	val := FuString("a")
	args := []FuObject{val}
	err := fn.CheckArgs(args)
	assert.Equal(t,
		"function bar() requires at least 2 arguments (got 1)", err.Error())

	// 2 args are good
	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// 3 args are good
	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// 4 args are good
	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// but 5 args is *right out*
	args = append(args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function bar() takes at most 4 arguments (got 5)", err.Error())
}

func Test_FuFunction_CheckArgs_unlimited(t *testing.T) {
	fn := NewVariadicFunction("println", 0, -1, nil)
	val := FuString("a")
	args := []FuObject{val}

	err := fn.CheckArgs(args)
	assert.Nil(t, err)

	args = append(args, val, val, val, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)
}
