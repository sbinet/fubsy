package runtime

import (
	"testing"
	"os"
	"reflect"
	"github.com/stretchrcom/testify/assert"
	"fubsy/testutils"
)

func Test_FuFileFinder_String(t *testing.T) {
	var ff FuObject
	ff = &FuFileFinder{includes: []string {"*.c", "**/*.h"}}
	assert.Equal(t, "<*.c **/*.h>", ff.String())
}

func Test_FuFileFinder_Expand_empty(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// no patterns, no files: of course we find nothing
	ff := &FuFileFinder{}
	assertExpand(t, []string {}, ff)

	// patterns, but no files: still nothing
	ff.includes = []string {"**/*.c", "include/*.h", "*/*.txt"}
	assertExpand(t, []string {}, ff)

	// no patterns, some files: still nothing
	ff.includes = ff.includes[0:0]
	mkdirs("lib1", "lib1/sub", "lib2", "include")
	touchfiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")
	assertExpand(t, []string {}, ff)
}

func Test_FuFileFinder_single_include(t *testing.T) {
	// punt on this, since FuFileFinder is just a stub for now
	return

	cleanup := testutils.Chtemp()
	defer cleanup()

	mkdirs("lib1", "lib1/sub", "lib2", "include")
	touchfiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")

	ff := &FuFileFinder{includes: []string {"*/*.c"}}
	assertExpand(t, []string {"lib1/foo.c"}, ff)

	ff.includes[0] = "**/*.c"
	assertExpand(t, []string {"lib1/foo.c", "lib1/sub/blah.c"}, ff)

	ff.includes[0] = "in?lu?e/*.h"
	assertExpand(t, []string {"include/bip.h", "include/bop.h"}, ff)

	ff.includes[0] = "inc*/?i*.h"
	assertExpand(t, []string {"include/bip.h"}, ff)
}

func mkdirs(dirs ...string) {
	for _, dir := range dirs {
		if err := os.Mkdir(dir, 0755); err != nil {
			panic(err)
		}
	}
}

func touchfiles(filenames ...string) {
	for _, fn := range filenames {
		file, err := os.Create(fn)
		if err != nil {
			panic(err)
		}
		file.Close()
	}
}

func assertExpand(t *testing.T, expect []string, ff *FuFileFinder) {
	actual, err := ff.Expand(nil)
	assert.Nil(t, err)

	expectobj := makeFuList(expect...)
	if !reflect.DeepEqual(expectobj, actual) {
		t.Errorf("FuFileFinder.find(): expected\n%v\nbut got\n%v",
			expectobj, actual)
	}
}
