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
}

func Test_FuString_Add_list(t *testing.T) {
	cmd := FuString("ls")
	args := FuList([]FuObject {FuString("-l"), FuString("-a"), FuString("foo")})
	result, err := cmd.Add(args)
	assert.Nil(t, err)
	assert.Equal(t, "[ls,-l,-a,foo]", result.String())
}


func Test_FuList_String(t *testing.T) {
	l := FuList([]FuObject{FuString("beep"), FuString("meep")})
	assert.Equal(t, "[beep,meep]", l.String())

	l = FuList([]FuObject{FuString("beep"), FuString(""), FuString("meep")})
	assert.Equal(t, "[beep,,meep]", l.String())
}
