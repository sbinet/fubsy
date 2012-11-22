package runtime

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
)

func Test_FuString_String(t *testing.T) {
	s := FuString("bip bop")
	assert.Equal(t, "bip bop", s.String())
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
	args := FuList([]FuObject {FuString("-l"), FuString("-a"), FuString("foo")})
	result, err := cmd.Add(args)
	assert.Nil(t, err)
	assert.Equal(t, "[ls,-l,-a,foo]", result.String())
}

func Test_FuString_Expand(t *testing.T) {
	input := FuString("meep meep!")
	output, err := input.Expand(nil)
	assert.Nil(t, err)
	assert.Equal(t, input, output)

	// not testing variable expansion because it's not implemented yet
}

func Test_FuList_String(t *testing.T) {
	l := FuList([]FuObject{FuString("beep"), FuString("meep")})
	assert.Equal(t, "[beep,meep]", l.String())

	l = FuList([]FuObject{FuString("beep"), FuString(""), FuString("meep")})
	assert.Equal(t, "[beep,,meep]", l.String())
}

func Test_FuList_Add_list(t *testing.T) {
	l1 := FuList([]FuObject {FuString("foo"), FuString("bar")})
	l2 := FuList([]FuObject {FuString("qux")})

	result, err := l1.Add(l2)
	expect := FuList([]FuObject {
		FuString("foo"), FuString("bar"), FuString("qux")})
	assert.Nil(t, err)
	assert.Equal(t, expect, result)

	result, err = l2.Add(l1)
	expect = FuList([]FuObject {
		FuString("qux"), FuString("foo"), FuString("bar")})
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Add_string(t *testing.T) {
	cmd := FuList([]FuObject {
		FuString("ls"), FuString("-la")})
	arg := FuString("stuff/")

	result, err := cmd.Add(arg)
	expect := FuList([]FuObject {
		FuString("ls"), FuString("-la"), FuString("stuff/")})
	assert.Nil(t, err)
	assert.Equal(t, expect, result)
}

func Test_FuList_Expand(t *testing.T) {
	//input := newFuList(FuString("gob"), FuString("mob"))
	input := makeFuList("gob", "mob")
	output, err := input.Expand(nil)
	assert.Nil(t, err)
	assert.Equal(t, input, output)
}

// Convert a variable number of strings to a FuList of FuString.
func makeFuList(strings ...string) FuList {
	result := make(FuList, len(strings))
	for i, s := range strings {
		result[i] = FuString(s)
	}
	return result
}

func newFuList(objects ...FuObject) FuList {
	return FuList(objects)
}
