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
		{"foo/**/bar", "foo/.*/bar$"},
		{"**/foo/bar", ".*/foo/bar$"},
		{"foo/bar/**", "foo/bar/.*$"},
		{"foo/**/bar/**/baz/**", "foo/.*/bar/.*/baz/.*$"},
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

func Test_splitPattern_no_recursive(t *testing.T) {
	patterns := []string {
		"",
		"foobar",
		"foo/b?r/*/blah/*.[ch]",
	}
	for _, pattern := range patterns {
		recursive, _, _, err := splitPattern(pattern)
		assert.False(t, recursive)
		assert.Nil(t, err)
	}




	// var prefix, tail string
	// var err error
	// recursive, prefix, tail, err = splitPattern("")
	// assert.Nil(t, err)
	// assert.True(t, prefix == "" && tail == "")

	// recursive, prefix, tail, err = splitPattern("foobar")
	// assert.Nil(t, err)
	// assert.True(t, prefix == "foobar" && tail == "")

	// recursive, prefix, tail, err = splitPattern("foo/b?r/*/blah/*.[ch]")
	// assert.Nil(t, err)
	// assert.True(t, prefix == "foo/b?r/*/blah/*.[ch]" && tail == "")
}

func Test_splitPattern_valid_recursive(t *testing.T) {
	tests := []struct {glob string; prefix string; tail string} {
		{"**/*.c", ".", "*.c"},
		{"**/foo/b?r/*.[ch]", ".", "foo/b?r/*.[ch]"},
		{"foo/**/*.c", "foo", "*.c"},
		{"f?o/*/**/?eep/*.[ch]", "f?o/*", "?eep/*.[ch]"},
	}

	for _, test := range tests {
		recursive, prefix, tail, err := splitPattern(test.glob)
		assert.True(t, recursive)
		assert.Nil(t, err)
		assert.Equal(t, test.prefix, prefix)
		assert.Equal(t, test.tail, tail)
	}
}

func Test_splitPattern_invalid(t *testing.T) {
	patterns := []string {
		"**",
		"**/",
		"foo/**",
		"foo/**/",
		"foo**/x",
		"foo/**x",
	}

	for _, pattern := range patterns {
		_, _, _, err := splitPattern(pattern)
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

func Test_FuFileFinder_Expand_single_include(t *testing.T) {
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

func Test_FuFileFinder_Expand_double_recursion(t *testing.T) {
	// Java programmers love this sort of insanely deep structure, and
	// Ant supports patterns with multiple occurences of "**" ... so I
	// guess Fubsy has to support them too!
	cleanup := testutils.Chtemp()
	defer cleanup()

	mkdirs(
		"app1/src/main/org/example/app1/subpkg",
		"app1/src/test/org/example/app1/subpkg",
		"misc/app2/src/main/org/example/app2",
		"misc/app3/src/main/org/example/app3",
		"misc/app3/src/test/org/example/app3",
		)
	touchfiles(
		"app1/src/main/org/example/app1/App1.java",
		"app1/src/main/org/example/app1/Util.java",
		"app1/src/main/org/example/app1/doc.txt",
		"app1/src/main/org/example/app1/subpkg/Stuff.java",
		"app1/src/main/org/example/app1/subpkg/MoreStuff.java",
		"app1/src/test/org/example/app1/StuffTest.java",
		"misc/app2/src/main/org/example/app2/App2.java",
		"misc/app3/src/main/org/example/app3/App3.java",
		"misc/app3/src/main/org/example/app3/Helpers.java",
		"misc/app3/src/test/org/example/app3/TestHelpers.java",
		"misc/app3/src/test/org/example/app3/testdata",
		)

	var ff *FuFileFinder
	var expect []string
	ff = NewFileFinder([]string {"**/test/**/*.java"})
	expect = []string {
		"app1/src/test/org/example/app1/StuffTest.java",
		"misc/app3/src/test/org/example/app3/TestHelpers.java",
	}
	assertExpand(t, expect, ff)

	ff = NewFileFinder([]string {"**/test/**/*"})
	expect = []string {
		"app1/src/test/org/example/app1/StuffTest.java",
		"misc/app3/src/test/org/example/app3/TestHelpers.java",
		"misc/app3/src/test/org/example/app3/testdata",
	}
	assertExpand(t, expect, ff)

	ff = NewFileFinder([]string {"**/test/**"})
	assertExpand(t, expect, ff)
}

func Test_FileFinder_Add(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()
	mkdirs("src", "include", "doc")
	touchfiles(
		"src/foo.c", "src/foo.h", "main.c", "include/bop.h",
		"doc.txt", "doc/stuff.txt", "doc/blahblah.rst")

	ff1 := NewFileFinder([]string {"**/*.c"})
	ff2 := NewFileFinder([]string {"doc/*.txt"})

	expect := makeFuList("main.c", "src/foo.c", "doc/stuff.txt")

	sum, err := ff1.Add(ff2)
	assert.Nil(t, err)
	actual, err := sum.Expand(nil)
	assert.Nil(t, err)
	assert.Equal(t, expect, actual)

	ff3 := NewFileFinder([]string {"*c*/?o?.h"})
	expect = append(expect, makeFuList("include/bop.h", "src/foo.h")...)
	sum, err = sum.Add(ff3)
	actual, err = sum.Expand(nil)
	assert.Nil(t, err)
	assert.Equal(t, expect, actual)

}

func mkdirs(dirs ...string) {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
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
