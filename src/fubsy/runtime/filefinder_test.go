package runtime

import (
	"testing"
	"fubsy/testutils"
)

func Test_FuFileFinder_String(t *testing.T) {
	ff := &FuFileFinder{includes: []string {"*.c", "**/*.h"}}
	testutils.AssertStrings(t, "<*.c **/*.h>", ff.String())
}
