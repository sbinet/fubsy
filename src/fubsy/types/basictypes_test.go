package types

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
)

func Test_FuString_String(t *testing.T) {
	s := FuString("bip bop")
	assert.Equal(t, "bip bop", s.String())
}

func Test_basictypes_Equal(t *testing.T) {
	s1 := FuString("foo")
	s2 := FuString("foo")
	s3 := FuString("bar")
	l1 := FuList([]FuObject {s1, s3})
	l2 := FuList([]FuObject {s2, s3})
	l3 := FuList([]FuObject {s3})

	// FuString.Equal is just like builtin string ==
	assert.True(t, s1.Equal(s1))
	assert.True(t, s1.Equal(s2))
	assert.False(t, s1.Equal(s3))

	// a FuString is never equal to a FuList
	assert.False(t, s1.Equal(l1))
	assert.False(t, s3.Equal(l3))
	assert.False(t, l3.Equal(s3))

	// FuList.Equal() compares list contents
	assert.True(t, l1.Equal(l1))
	assert.True(t, l1.Equal(l2))
	assert.False(t, l1.Equal(l3))
}

func Test_FuString_Add_strings(t *testing.T) {
	s1 := FuString("hello")
	s2 := FuString("world")
	var result FuObject
	var err error

	// s1 + s1
	result, err = s1.Add(s1)
	assert.Nil(t, err)
	assert.Equal(t, "hellohello", result.(FuString))

	// s1 + s2
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworld", result.(FuString))

	// s1 + s2 + s1 + s2
	// (equivalent to ((s1.Add(s2)).Add(s1)).Add(s2), except we have
	// to worry about error handling)
	result, err = s1.Add(s2)
	assert.Nil(t, err)
	result, err = result.Add(s1)
	assert.Nil(t, err)
	result, err = result.Add(s2)
	assert.Nil(t, err)
	assert.Equal(t, "helloworldhelloworld", result.(FuString))

	// neither s1 nor s2 is affected by all this adding
	assert.Equal(t, "hello", string(s1))
	assert.Equal(t, "world", string(s2))
}

func Test_FuString_Add_list(t *testing.T) {
	cmd := FuString("ls")
	args := makeFuList("-l", "-a", "foo")
	result, err := cmd.Add(args)
	assert.Nil(t, err)
	assert.Equal(t, "[ls,-l,-a,foo]", result.String())
}

func Test_FuString_Expand(t *testing.T) {
	input := FuString("meep meep!")
	output, err := input.Expand()
	assert.Nil(t, err)
	assert.Equal(t, input, output)

	// not testing variable expansion because it's not implemented yet
}

func Test_FuList_String(t *testing.T) {
	l := makeFuList("beep", "meep")
	assert.Equal(t, "[beep,meep]", l.String())

	l = makeFuList("beep", "", "meep")
	assert.Equal(t, "[beep,,meep]", l.String())
}

func Test_FuList_Add_list(t *testing.T) {
	l1 := makeFuList("foo", "bar")
	l2 := makeFuList("qux")

	result, err := l1.Add(l2)
	expect := makeFuList("foo", "bar", "qux")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)

	result, err = l2.Add(l1)
	expect = makeFuList("qux", "foo", "bar")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Add_string(t *testing.T) {
	cmd := makeFuList("ls", "-la")
	arg := FuString("stuff/")

	result, err := cmd.Add(arg)
	expect := makeFuList("ls", "-la", "stuff/")
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Expand(t *testing.T) {
	input := makeFuList("gob", "mob")
	output, err := input.Expand()
	assert.Nil(t, err)
	assert.Equal(t, input, output)
}
