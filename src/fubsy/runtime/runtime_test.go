// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"bytes"
	"testing"
	//"fmt"
	//"reflect"

	"github.com/stretchrcom/testify/assert"

	"fubsy/build"
	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/types"
)

func Test_Runtime_runMainPhase_missing(t *testing.T) {
	// invalid: a script with no main phase
	filename := "test.fubsy"
	script := "" +
		"import meep\n" +
		"plunk {\n" +
		"}\n"
	rt := parseScript(t, filename, script)
	errors := rt.runMainPhase()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "test.fubsy:4: no main phase defined", errors[0].Error())
}

func Test_Runtime_runMainPhase_valid(t *testing.T) {
	script := "" +
		"main {\n" +
		"  src = \"foo.c\"\n" +
		"  \"foo\": src {\n" +
		"    \"cc -o $TARGET $src\"\n" +
		"  }\n" +
		"}\n"
	rt := parseScript(t, "test.fubsy", script)
	errors := rt.runMainPhase()
	assert.Equal(t, 0, len(errors))
	val, ok := rt.stack.Lookup("src")
	assert.True(t, ok)
	assert.Equal(t, types.FuString("foo.c"), val)
	assert.NotNil(t, rt.dag)

	// this seems *awfully* detailed and brittle, but DAG doesn't
	// provide a good way to query what's in it (yet...)
	expect := "" +
		"0000: FileNode foo (state UNKNOWN)\n" +
		"  action: cc -o $TARGET $src\n" +
		"  parents:\n" +
		"    0001: foo.c\n" +
		"0001: FileNode foo.c (state UNKNOWN)\n"
	var buf bytes.Buffer
	rt.dag.Dump(&buf, "")
	assert.Equal(t, expect, buf.String())
}

func Test_Runtime_runMainPhase_error(t *testing.T) {
	// runtime error evaluating a build rule
	script := "" +
		"main {\n" +
		"  \"foo.jar\": bogus {\n" +
		"    \"javac && jar\"\n" +
		"  }\n" +
		"}\n"
	rt := parseScript(t, "test.fubsy", script)
	errors := rt.runMainPhase()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"test.fubsy:2: name not defined: 'bogus'",
		errors[0].Error())
}

func parseScript(t *testing.T, filename string, content string) *Runtime {
	ast, errors := dsl.ParseString(filename, content)
	assert.Equal(t, 0, len(errors)) // syntax must be good
	return NewRuntime(build.BuildOptions{}, filename, ast)
}

func Test_nodify(t *testing.T) {
	sval1 := types.FuString("hello.txt")
	sval2 := types.FuString("foo.c")
	lval1 := types.FuList([]types.FuObject{sval1, sval2})
	finder1 := dag.NewFinderNode("*.c", "*.h")
	finder2 := dag.NewFinderNode("**/*.java")

	rt := NewRuntime(build.BuildOptions{}, "", nil)
	nodes := rt.nodify(sval1)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "hello.txt", nodes[0].(*dag.FileNode).String())

	nodes = rt.nodify(lval1)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, "hello.txt", nodes[0].(*dag.FileNode).String())
	assert.Equal(t, "foo.c", nodes[1].(*dag.FileNode).String())

	nodes = rt.nodify(finder1)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "<*.c *.h>", nodes[0].(*dag.FinderNode).String())

	findersum, err := finder1.Add(finder2)
	assert.Nil(t, err)
	nodes = rt.nodify(findersum)
	if len(nodes) == 2 {
		assert.Equal(t, "<*.c *.h>", nodes[0].(*dag.FinderNode).String())
		assert.Equal(t, "<**/*.java>", nodes[1].(*dag.FinderNode).String())
	} else {
		t.Errorf("expected nodify(%s) to return 2 nodes, but got %d: %v",
			findersum, len(nodes), nodes)
	}
}
