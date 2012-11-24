package runtime

import (
	"testing"
	"os"
	"reflect"
	"regexp"
	"github.com/stretchrcom/testify/assert"
	"fubsy/testutils"
)

func Test_translateGlob(t *testing.T) {
	tests := []struct {glob string; re string} {
		{"", "$"},
		{"foo", "foo$"},
		{"foo/bar", "foo/bar$"},

		{"foo?bar", "foo[^/]bar$"},
		{"*.c", "[^/]*\\.c$"},
		{"foo[abc]", "foo[abc]$"},
		{"foo[a-m]*.bop", "foo[a-m][^/]*\\.bop$"},
	}

	for _, pair := range tests {
		actual, err := translateGlob(pair.glob)
		assert.Nil(t, err)
		assert.Equal(t, pair.re, actual)
	}

	// make sure one of those regexps actually works as intended
	pat, err := translateGlob("foo[a-m]/p*g/*.[ch]")
	assert.Nil(t, err)
	re, err := regexp.Compile("^" + pat)
	assert.Nil(t, err)
	match := []string {
		"foom/pong/bop.c",
		"foog/pig/abc.c",
		"foog/pig/a.c.-af#@0(.h",
		"foob/pg/a_b&.c",
	}
	for _, fn := range match {
		assert.Equal(t, fn, re.FindString(fn))
	}

	nomatch := []string {
		"foo/pong/bop.c",
		"foom/pongx/bop.c",
		"foom/pong/bop.cpp",
		"foom/pong/bop.c/x",
		"fooz/pong/bop.c",
		"foom/pg/bop.d",
	}
	for _, fn := range nomatch {
		assert.Equal(t, "", re.FindString(fn))
	}
}

func Test_findRecursive_no_recursive(t *testing.T) {
	var prefix, tail string
	var err error
	prefix, tail, err = findRecursive("")
	assert.Nil(t, err)
	assert.True(t, prefix == "" && tail == "")

	prefix, tail, err = findRecursive("foobar")
	assert.Nil(t, err)
	assert.True(t, prefix == "foobar" && tail == "")

	prefix, tail, err = findRecursive("foo/b?r/*/blah/*.[ch]")
	assert.Nil(t, err)
	assert.True(t, prefix == "foo/b?r/*/blah/*.[ch]" && tail == "")
}

func Test_findRecursive_valid_recursive(t *testing.T) {
	var prefix, tail string
	var err error
	prefix, tail, err = findRecursive("**/*.c")
	assert.Nil(t, err)
	assert.Equal(t, "", prefix)
	assert.Equal(t, "*.c", tail)

	prefix, tail, err = findRecursive("**/foo/b?r/*.[ch]")
	assert.Nil(t, err)
	assert.Equal(t, "", prefix)
	assert.Equal(t, "foo/b?r/*.[ch]", tail)

	prefix, tail, err = findRecursive("foo/**/*.c")
	assert.Nil(t, err)
	assert.Equal(t, "foo", prefix)
	assert.Equal(t, "*.c", tail)

	prefix, tail, err = findRecursive("f?o/*/**/?eep/*.[ch]")
	assert.Nil(t, err)
	assert.Equal(t, "f?o/*", prefix)
	assert.Equal(t, "?eep/*.[ch]", tail)
}

func Test_findRecursive_invalid(t *testing.T) {
	patterns := []string {
		"**",
		"**/",
		"foo/**",
		"foo/**/",
		"foo**/x",
		"foo/**x",
	}

	for _, pattern := range patterns {
		_, _, err := findRecursive(pattern)
		assert.NotNil(t, err)
	}
}


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
	cleanup := testutils.Chtemp()
	defer cleanup()

	mkdirs("lib1", "lib1/sub", "lib2", "include")
	touchfiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")

	ff := &FuFileFinder{includes: []string {"*/*.c"}}
	assertExpand(t, []string {"lib1/foo.c"}, ff)

	ff.includes[0] = "**/*.c"
	assertExpand(t, []string {"lib1/foo.c", "lib1/sub/blah.c"}, ff)
	return

	ff.includes[0] = "l?b?/**/*.c"
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
		t.Errorf("FuFileFinder.Expand(): includes=%v: " +
			"expected\n%v\nbut got\n%v",
			ff.includes, expectobj, actual)
	}
}
