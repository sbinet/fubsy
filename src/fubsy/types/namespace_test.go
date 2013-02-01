// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_ValueMap_basics(t *testing.T) {
	ns := NewValueMap()
	val, ok := ns.Lookup("foo")
	assert.False(t, ok)
	val, ok = ns.Lookup("bar")
	assert.False(t, ok)

	ns.Assign("foo", FuString("blurp"))
	val, ok = ns.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, "blurp", val.ValueString())
	val, ok = ns.Lookup("bar")
	assert.False(t, ok)
}

func Test_ValueStack_Lookup(t *testing.T) {
	empty := NewValueStack()
	val, ok := empty.Lookup("foo")
	assert.False(t, ok)

	list1 := FuList([]FuObject{FuString("ding"), FuString("dong")})
	list2 := FuList([]FuObject{FuString("shadow")})

	ns0 := NewValueMap()
	ns0.Assign("foo", FuString("hello"))
	ns0.Assign("bar", list1)
	ns1 := NewValueMap()
	ns1.Assign("foo", list2)
	stack := NewValueStack(ns0, ns1)

	val, ok = stack.Lookup("foo")
	assert.True(t, ok)
	assert.True(t, val.Equal(list2))
	val, ok = stack.Lookup("bar")
	assert.True(t, ok)
	assert.True(t, val.Equal(list1))
	val, ok = stack.Lookup("x")
	assert.False(t, ok)
}

func Test_ValueStack_Assign(t *testing.T) {
	ns0 := NewValueMap()
	stack := NewValueStack()
	assert.Equal(t, 0, len(stack))
	stack.Push(ns0)
	assert.Equal(t, 1, len(stack))

	val1 := FuString("hello")
	val2 := FuString("world")
	val3 := FuString("fnord")

	stack.Assign("foo", val1)
	val, ok := stack.Lookup("foo")
	assert.True(t, ok)
	assert.True(t, val.Equal(val1))
	stack.Assign("foo", val2)
	val, ok = stack.Lookup("foo")
	assert.True(t, ok)
	assert.True(t, val.Equal(val2))

	ns1 := NewValueMap()
	stack.Push(ns1)
	assert.Equal(t, 2, len(stack))
	stack.Assign("bar", val1)
	stack.Assign("foo", val3)

	// make sure we can get the new values out of the stack
	val, ok = stack.Lookup("bar")
	assert.True(t, val.Equal(val1))
	val, ok = stack.Lookup("foo")
	assert.True(t, val.Equal(val3))

	// peek under the hood and make sure each one was added to the
	// right namespace: bar is a new name, so it will be in the
	// innermost (last) namespace
	val, ok = ns1.Lookup("bar")
	assert.True(t, val.Equal(val1))
	val, ok = ns0.Lookup("bar")
	assert.False(t, ok)

	// foo already existed in an enclosing namespace, so Assign()
	// replaced it there
	val, ok = ns0.Lookup("foo")
	assert.True(t, val.Equal(val3))
	val, ok = ns1.Lookup("foo")
	assert.False(t, ok)
}
