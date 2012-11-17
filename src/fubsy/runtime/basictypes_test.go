package runtime

import (
	"testing"
	"fubsy/testutils"
)

func Test_FuString_String(t *testing.T) {
	s := FuString("bip bop")
	testutils.AssertStrings(t, "bip bop", s.String())
}


func Test_FuList_String(t *testing.T) {
	l := FuList([]FuObject{FuString("beep"), FuString("meep")})
	testutils.AssertStrings(t, "[beep,meep]", l.String())

	l = FuList([]FuObject{FuString("beep"), FuString(""), FuString("meep")})
	testutils.AssertStrings(t, "[beep,,meep]", l.String())
}
