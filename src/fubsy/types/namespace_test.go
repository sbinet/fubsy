package types

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
)

func Test_ValueMap_basics(t *testing.T) {
	ns := NewValueMap()
	assert.Nil(t, ns.Lookup("foo"))
	assert.Nil(t, ns.Lookup("bar"))

	ns.Assign("foo", FuString("blurp"))
	assert.Equal(t, "blurp", ns.Lookup("foo").String())
	assert.Nil(t, ns.Lookup("bar"))
}

func Test_ValueStack_Lookup(t *testing.T) {
	empty := NewValueStack()
	assert.Nil(t, empty.Lookup("foo"))

	list1 := FuList([]FuObject {FuString("ding"), FuString("dong")})
	list2 := FuList([]FuObject {FuString("shadow")})

	ns0 := NewValueMap()
	ns0.Assign("foo", FuString("hello"))
	ns0.Assign("bar", list1)
	ns1 := NewValueMap()
	ns1.Assign("foo", list2)
	stack := NewValueStack(ns0, ns1)

	assert.True(t, stack.Lookup("foo").Equal(list2))
	assert.True(t, stack.Lookup("bar").Equal(list1))
	assert.Nil(t, stack.Lookup("x"))
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
	assert.True(t, stack.Lookup("foo").Equal(val1))
	stack.Assign("foo", val2)
	assert.True(t, stack.Lookup("foo").Equal(val2))

	ns1 := NewValueMap()
	stack.Push(ns1)
	assert.Equal(t, 2, len(stack))
	stack.Assign("bar", val1)
	stack.Assign("foo", val3)

	// make sure we can get the new values out of the stack
	assert.True(t, stack.Lookup("bar").Equal(val1))
	assert.True(t, stack.Lookup("foo").Equal(val3))

	// peek under the hood and make sure each one was added to the
	// right namespace: bar is a new name, so it will be in the
	// innermost (last) namespace
	assert.True(t, ns1.Lookup("bar").Equal(val1))
	assert.Nil(t, ns0.Lookup("bar"))

	// foo already existed in an enclosing namespace, so Assign()
	// replaced it there
	assert.True(t, ns0.Lookup("foo").Equal(val3))
	assert.Nil(t, ns1.Lookup("foo"))
}
