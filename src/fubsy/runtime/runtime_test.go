// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"bytes"
	"os"
	"testing"
	//"fmt"
	//"reflect"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/testutils"
	"fubsy/types"
)

func Test_Runtime_runMainPhase_missing(t *testing.T) {
	// invalid: a script with no main phase
	filename := "test.fubsy"
	script := "" +
		"import meep\n" +
		"plunk {\n" +
		"}\n"
	// ast, err := dsl.ParseString(filename, script)
	// assert.Equal(t, 0, len(err)) // syntax is fine
	// rt := NewRuntime(filename, ast)
	rt := parseScript(t, filename, script)
	errors := rt.runMainPhase()
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "test.fubsy:1-3: no main phase defined", errors[0].Error())
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
	rt.dag.Dump(os.Stdout)

	// this seems *awfully* detailed and brittle, but DAG doesn't
	// provide a good way to query what's in it (yet...)
	expect := "" +
		"0000: foo (*dag.FileNode, UNKNOWN)\n" +
		"  action: cc -o $TARGET $src\n" +
		"  parents:\n" +
		"    0001: foo.c\n" +
		"0001: foo.c (*dag.FileNode, UNKNOWN)\n"
	var buf bytes.Buffer
	rt.dag.Dump(&buf)
	assert.Equal(t, expect, buf.String())
}

func Test_Runtime_runMainPhase_error(t *testing.T) {
	// runtime error evaluating a build rule (cannot add string to filefinder)
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
		"test.fubsy:2: undefined variable 'bogus'",
		errors[0].Error())
}

func parseScript(t *testing.T, filename string, content string) *Runtime {
	ast, errors := dsl.ParseString(filename, content)
	assert.Equal(t, 0, len(errors)) // syntax must be good
	return NewRuntime(filename, ast)
}

func Test_Runtime_assign(t *testing.T) {
	// AST for a = "foo"
	node := dsl.NewASTAssignment("a", stringnode("foo"))
	rt := &Runtime{}
	ns := types.NewValueMap()

	rt.assign(node, ns)
	expect := types.FuString("foo")
	assertIn(t, ns, "a", expect)
}

// evaluate simple expressions (no operators)
func Test_Runtime_evaluate_simple(t *testing.T) {
	// the expression "meep" evaluates to the string "meep"
	var expect types.FuObject
	snode := stringnode("meep")
	rt := NewRuntime("", nil)
	expect = types.FuString("meep")
	assertEvaluateOK(t, rt, expect, snode)

	// the expression foo evaluates to the string "meep" if foo is set
	// to that string in the global ValueMap
	rt.globals.Assign("foo", expect)
	nnode := dsl.NewASTName("foo")
	assertEvaluateOK(t, rt, expect, nnode)

	// ... and to an error if the variable is not defined
	nnode = dsl.NewASTName("boo")
	assertEvaluateFail(t, rt, "undefined variable 'boo'", nnode)

	// expression <*.c blah> evaluates to a FileFinder with two
	// include patterns
	patterns := []string{"*.c", "blah"}
	flnode := dsl.NewASTFileList(patterns)
	expect = types.NewFileFinder([]string{"*.c", "blah"})
	assertEvaluateOK(t, rt, expect, flnode)
}

func stringnode(value string) *dsl.ASTString {
	// NewASTString takes a token, which comes quoted
	value = "\"" + value + "\""
	return dsl.NewASTString(value)
}

func Test_nodify(t *testing.T) {
	sval1 := types.FuString("hello.txt")
	sval2 := types.FuString("foo.c")
	lval1 := types.FuList([]types.FuObject{sval1, sval2})
	ff1 := types.NewFileFinder([]string{"*.c", "*.h"})
	ff2 := types.NewFileFinder([]string{"**/*.java"})

	rt := NewRuntime("", nil)
	nodes := rt.nodify(sval1)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "hello.txt", nodes[0].(*dag.FileNode).String())

	nodes = rt.nodify(lval1)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, "hello.txt", nodes[0].(*dag.FileNode).String())
	assert.Equal(t, "foo.c", nodes[1].(*dag.FileNode).String())

	nodes = rt.nodify(ff1)
	assert.Equal(t, 1, len(nodes))
	assert.Equal(t, "<*.c *.h>", nodes[0].(*dag.GlobNode).String())

	ffsum, err := ff1.Add(ff2)
	assert.Nil(t, err)
	nodes = rt.nodify(ffsum)
	assert.Equal(t, 2, len(nodes))
	assert.Equal(t, "<*.c *.h>", nodes[0].(*dag.GlobNode).String())
	assert.Equal(t, "<**/*.java>", nodes[1].(*dag.GlobNode).String())
}

func assertIn(t *testing.T, ns types.ValueMap, name string, expect types.FuObject) {
	if actual, ok := ns[name]; ok {
		if actual != expect {
			t.Errorf("expected %#v, but got %#v", expect, actual)
		}
	} else {
		t.Errorf("expected to find name '%s' in namespace", name)
	}
}

func assertEvaluateOK(
	t *testing.T,
	rt *Runtime,
	expect types.FuObject,
	input dsl.ASTExpression) {

	obj, err := rt.evaluate(input)
	assert.Nil(t, err)

	// need to use DeepEqual() to handle (e.g.) slices inside structs
	//if !reflect.DeepEqual(expect, obj) {
	if !expect.Equal(obj) {
		t.Errorf("expected\n%#v\nbut got\n%#v", expect, obj)
	}
}

func assertEvaluateFail(
	t *testing.T,
	rt *Runtime,
	expecterr string,
	input dsl.ASTExpression) {

	obj, err := rt.evaluate(input)
	testutils.AssertError(t, expecterr, err)
	if obj != nil {
		t.Errorf("expected obj == nil, but got %#v", obj)
	}
}
