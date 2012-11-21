package runtime

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
)

func Test_FuString_String(t *testing.T) {
	s := FuString("bip bop")
	assert.Equal(t, "bip bop", s.String())
}


func Test_FuList_String(t *testing.T) {
	l := FuList([]FuObject{FuString("beep"), FuString("meep")})
	assert.Equal(t, "[beep,meep]", l.String())

	l = FuList([]FuObject{FuString("beep"), FuString(""), FuString("meep")})
	assert.Equal(t, "[beep,,meep]", l.String())
}
