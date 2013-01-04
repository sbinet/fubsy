// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"regexp"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_FinderNode_String(t *testing.T) {
	var finder types.FuObject
	finder = &FinderNode{includes: []string{"*.c", "**/*.h"}}
	assert.Equal(t, "<*.c **/*.h>", finder.String())
}

func Test_FinderNode_CommandString(t *testing.T) {
	var finder types.FuObject
	finder = &FinderNode{includes: []string{"*.c", "blurp/blop", "**/*.h"}}
	assert.Equal(t, "'*.c' blurp/blop '**/*.h'", finder.CommandString())
}

func Test_FinderNode_Equal(t *testing.T) {
	finder1 := NewFinderNode("*.c", "*.h")
	finder2 := NewFinderNode("*.c", "*.h")
	finder3 := NewFinderNode("*.h", "*.c")

	assert.True(t, finder1.Equal(finder1))
	assert.True(t, finder1.Equal(finder2))
	assert.False(t, finder1.Equal(finder3))
}

func Test_FinderNode_Add_Expand(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()
	testutils.TouchFiles(
		"src/foo.c", "src/foo.h", "main.c", "include/bop.h",
		"doc.txt", "doc/stuff.txt", "doc/blahblah.rst")

	finder1 := NewFinderNode("**/*.c")
	finder2 := NewFinderNode("doc/*.txt")

	// sum = <**/*.c> + <doc/*.txt>
	expect := []string{
		"main.c", "src/foo.c", "doc/stuff.txt"}
	sum, err := finder1.Add(finder2)
	assert.Nil(t, err)
	assertExpand(t, expect, sum)

	// sum = sum + <"*c*/?o?.h">
	finder3 := NewFinderNode("*c*/?o?.h")
	expect = append(expect, "include/bop.h", "src/foo.h")
	sum, err = sum.Add(finder3)
	assert.Nil(t, err)
	assertExpand(t, expect, sum)

	// simulate <doc/*.txt> + (<*c*/?o?.h> + <**/*.c>)
	// (same as before, just different order)
	expect = []string{
		"include/bop.h", "src/foo.h", "main.c", "src/foo.c"}
	sum, err = finder3.Add(finder1)
	assert.Nil(t, err)
	assertExpand(t, expect, sum)

	expect = append([]string{"doc/stuff.txt"}, expect...)
	sum, err = finder2.Add(sum)
	assert.Nil(t, err)
	assertExpand(t, expect, sum)

	// <**/*.c> + "urgh"
	expect = []string{
		"main.c", "src/foo.c", "urgh"}
	sum, err = finder1.Add(types.FuString("urgh"))
	assert.Nil(t, err)
	assertExpand(t, expect, sum)

	// <**/*.c> + ["a", "b", "c"]
	expect = []string{
		"main.c", "src/foo.c", "a", "b", "c"}
	sum, err = finder1.Add(types.MakeFuList("a", "b", "c"))
	assert.Nil(t, err)
	assertExpand(t, expect, sum)
}

// hmmmm: interface-wise, this tests that FinderNode.Add() returns an
// object whose CommandString() behaves sensibly... but in
// implementation terms, it's really a test of FuList.CommandString()
func Test_FinderNode_Add_CommandString(t *testing.T) {
	finder1 := NewFinderNode("*.c", "*.h")
	finder2 := NewFinderNode("doc/???.txt")
	finder3 := NewFinderNode()

	sum1, err := finder1.Add(finder2)
	assert.Nil(t, err)
	assert.Equal(t, "'*.c' '*.h' 'doc/???.txt'", sum1.CommandString())

	sum2, err := finder3.Add(sum1)
	assert.Nil(t, err)
	assert.Equal(t, "'*.c' '*.h' 'doc/???.txt'", sum2.CommandString())

	assert.False(t, sum1.Equal(sum2))

	sum2b, err := finder3.Add(sum1)
	assert.Nil(t, err)
	assert.True(t, sum2.Equal(sum2b))

	// This is a silly thing to do, and perhaps we should filter out
	// the duplicate patterns... but I don't think so. If the user
	// constructs something silly, we do something silly.
	sum3, err := sum1.Add(sum2)
	assert.Nil(t, err)
	assert.Equal(t,
		"'*.c' '*.h' 'doc/???.txt' '*.c' '*.h' 'doc/???.txt'",
		sum3.CommandString())
}

func Test_FinderNode_Expand_empty(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// no patterns, no files: of course we find nothing
	finder := &FinderNode{}
	assertExpand(t, []string{}, finder)

	// patterns, but no files: still nothing
	finder.includes = []string{"**/*.c", "include/*.h", "*/*.txt"}
	assertExpand(t, []string{}, finder)

	// no patterns, some files: still nothing
	finder.includes = finder.includes[0:0]
	testutils.TouchFiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")
	assertExpand(t, []string{}, finder)
}

func Test_FinderNode_Expand_single_include(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.TouchFiles(
		"lib1/foo.c", "lib1/sub/blah.c", "include/bop.h", "include/bip.h")

	finder := NewFinderNode("*/*.c")
	assertExpand(t, []string{"lib1/foo.c"}, finder)

	finder.includes[0] = "**/*.c"
	assertExpand(t, []string{"lib1/foo.c", "lib1/sub/blah.c"}, finder)

	finder.includes[0] = "l?b?/**/*.c"
	assertExpand(t, []string{"lib1/foo.c", "lib1/sub/blah.c"}, finder)

	finder.includes[0] = "in?lu?e/*.h"
	assertExpand(t, []string{"include/bip.h", "include/bop.h"}, finder)

	finder.includes[0] = "inc*/?i*.h"
	assertExpand(t, []string{"include/bip.h"}, finder)
}

func Test_FinderNode_Expand_double_recursion(t *testing.T) {
	// Java programmers love this sort of insanely deep structure, and
	// Ant supports patterns with multiple occurences of "**" ... so I
	// guess Fubsy has to support them too!
	cleanup := testutils.Chtemp()
	defer cleanup()

	testutils.TouchFiles(
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

	var finder *FinderNode
	var expect []string
	finder = NewFinderNode("**/test/**/*.java")
	expect = []string{
		"app1/src/test/org/example/app1/StuffTest.java",
		"misc/app3/src/test/org/example/app3/TestHelpers.java",
	}
	assertExpand(t, expect, finder)

	finder = NewFinderNode("**/test/**/*")
	expect = []string{
		"app1/src/test/org/example/app1/StuffTest.java",
		"misc/app3/src/test/org/example/app3/TestHelpers.java",
		"misc/app3/src/test/org/example/app3/testdata",
	}
	assertExpand(t, expect, finder)

	finder = NewFinderNode("**/test/**")
	assertExpand(t, expect, finder)
}

func assertExpand(t *testing.T, expect []string, obj types.FuObject) {
	ns := types.NewValueMap()
	actualobj, err := obj.Expand(ns)
	assert.Nil(t, err)

	// convert FuList of FileNode to slice of string
	actualstr := make([]string, len(actualobj.List()))
	for i, obj := range actualobj.List() {
		actualstr[i] = obj.String()
	}
	assert.Equal(t, expect, actualstr)
}

func Test_MakeFinderNode(t *testing.T) {
	dag := NewDAG()
	node0 := MakeFinderNode(dag, "**/*.java")
	node1 := MakeFinderNode(dag, "doc/*/*.html")
	assert.Equal(t, node0.Name(), dag.nodes[0].Name())
	assert.Equal(t, node1.Name(), dag.nodes[1].Name())

	// correctly reuse existing entries
	dupnode := MakeFinderNode(dag, "**/*.java")
	assert.Equal(t, dag.nodes[0].Name(), dupnode.Name())

	assert.Equal(t, "<**/*.java>", node0.String())
	assert.Equal(t, "<doc/*/*.html>", node1.String())
}

func Test_translateGlob(t *testing.T) {
	tests := []struct {
		glob string
		re   string
	}{
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
	match := []string{
		"foom/pong/bop.c",
		"foog/pig/abc.c",
		"foog/pig/a.c.-af#@0(.h",
		"foob/pg/a_b&.c",
	}
	for _, fn := range match {
		assert.Equal(t, fn, re.FindString(fn))
	}

	nomatch := []string{
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

func Test_translateGlob_error(t *testing.T) {
	_, err := translateGlob("foo[a-f*.c")
	assert.Equal(t, "unterminated character range", err.Error())
}

func Test_splitPattern_no_recursive(t *testing.T) {
	patterns := []string{
		"",
		"foobar",
		"foo/b?r/*/blah/*.[ch]",
	}
	for _, pattern := range patterns {
		recursive, _, _, err := splitPattern(pattern)
		assert.False(t, recursive)
		assert.Nil(t, err)
	}
}

func Test_splitPattern_valid_recursive(t *testing.T) {
	tests := []struct {
		glob   string
		prefix string
		tail   string
	}{
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
	patterns := []string{
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
