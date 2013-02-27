// Copyright © 2012-2013, Greg Ward. All rights reserved.
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
	val := MakeFuString("a")
	args := MakeBasicArgs(nil, []FuObject{}, nil)
	fn := NewFixedFunction("meep", 0, nil)

	err := fn.CheckArgs(args)
	assert.Nil(t, err)

	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function meep() takes no arguments (got 1)", err.Error())

	fn = NewFixedFunction("foo", 2, nil)
	args.args = args.args[:0]
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 0)", err.Error())

	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 1)", err.Error())

	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function foo() takes exactly 2 arguments (got 3)", err.Error())
}

func Test_FuFunction_CheckArgs_minmax(t *testing.T) {
	fn := NewVariadicFunction("bar", 2, 4, nil)
	val := MakeFuString("a")
	args := MakeBasicArgs(nil, []FuObject{val}, nil)
	err := fn.CheckArgs(args)
	assert.Equal(t,
		"function bar() requires at least 2 arguments (got 1)", err.Error())

	// 2 args are good
	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// 3 args are good
	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// 4 args are good
	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)

	// but 5 args is *right out*
	args.args = append(args.args, val)
	err = fn.CheckArgs(args)
	assert.Equal(t,
		"function bar() takes at most 4 arguments (got 5)", err.Error())
}

func Test_FuFunction_CheckArgs_unlimited(t *testing.T) {
	fn := NewVariadicFunction("println", 0, -1, nil)
	val := MakeFuString("a")
	args := MakeBasicArgs(nil, []FuObject{val}, nil)

	err := fn.CheckArgs(args)
	assert.Nil(t, err)

	args.args = append(args.args, val, val, val, val)
	err = fn.CheckArgs(args)
	assert.Nil(t, err)
}

func Test_BasicArgs(t *testing.T) {
	var args BasicArgs
	var _ ArgSource = args

	args.args = MakeStringList("foo", "bar").List()
	assert.Nil(t, args.Receiver())
	assert.Equal(t, args.args, args.Args())
}
