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

func Test_FuFileFinder_empty(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// no patterns, no files: of course we find nothing
	ff := &FuFileFinder{}
	assertFind(t, []string {}, ff)

	// patterns, but no files: still nothing
	ff.includes = []string {"**/*.c", "include/*.h", "*/*.txt"}
	assertFind(t, []string {}, ff)

	// no patterns, some files: still nothing
	ff.includes = ff.includes[0:0]
	mkdirs("lib1", "lib1/sub", "lib2", "include")
	touchfiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")
	assertFind(t, []string {}, ff)
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
	assertFind(t, []string {"lib1/foo.c"}, ff)

	ff.includes[0] = "**/*.c"
	assertFind(t, []string {"lib1/foo.c", "lib1/sub/blah.c"}, ff)

	ff.includes[0] = "in?lu?e/*.h"
	assertFind(t, []string {"include/bip.h", "include/bop.h"}, ff)

	ff.includes[0] = "inc*/?i*.h"
	assertFind(t, []string {"include/bip.h"}, ff)
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

func assertFind(t *testing.T, expect []string, ff *FuFileFinder) {
	actual := ff.find()
	if !reflect.DeepEqual(expect, actual) {
		t.Errorf("FuFileFinder.find(): expected\n%v\nbut got\n%v",
			expect, actual)
	}
}
