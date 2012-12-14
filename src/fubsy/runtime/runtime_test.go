// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"testing"
	//"reflect"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dsl"
	"fubsy/testutils"
	"fubsy/types"
)

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
