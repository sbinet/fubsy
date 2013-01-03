// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/types"
)

func Test_assign(t *testing.T) {
	// AST for a = "foo"
	node := dsl.NewASTAssignment("a", stringnode("foo"))
	ns := types.NewValueMap()

	errs := assign(ns, node)
	assert.Equal(t, 0, len(errs))
	expect := types.FuString("foo")
	assertIn(t, ns, "a", expect)

	// AST for a = foo (another variable, to provoke an error)
	node = dsl.NewASTAssignment("b", dsl.NewASTName("foo"))
	errs = assign(ns, node)
	assert.Equal(t, "name not defined: 'foo'", errs[0].Error())
	_, ok := ns.Lookup("b")
	assert.False(t, ok)
}

// evaluate simple expressions (no operators)
func Test_evaluate_simple(t *testing.T) {
	// the expression "meep" evaluates to the string "meep"
	var expect types.FuObject
	snode := stringnode("meep")
	ns := types.NewValueMap()
	expect = types.FuString("meep")
	assertEvaluateOK(t, ns, expect, snode)

	// the expression foo evaluates to the string "meep" if foo is set
	// to that string
	ns.Assign("foo", expect)
	nnode := dsl.NewASTName("foo")
	assertEvaluateOK(t, ns, expect, nnode)

	// ... and to an error if the variable is not defined
	location := dsl.NewStubLocation("hello, sailor")
	nnode = dsl.NewASTName("boo", location)
	assertEvaluateFail(t, ns, "hello, sailor: name not defined: 'boo'", nnode)

	// expression <*.c blah> evaluates to a FinderNode with two
	// include patterns
	patterns := []string{"*.c", "blah"}
	fnode := dsl.NewASTFileFinder(patterns)
	expect = dag.NewFinderNode([]string{"*.c", "blah"})
	assertEvaluateOK(t, ns, expect, fnode)
}

// evaluate more complex expressions
func Test_evaluate_complex(t *testing.T) {
	// a + b evaluates to various things, depending on the value
	// of those two variables
	addnode := dsl.NewASTAdd(
		dsl.NewASTName("a", dsl.NewStubLocation("loc1")),
		dsl.NewASTName("b", dsl.NewStubLocation("loc2")))

	// case 1: two strings just get concatenated
	ns := types.NewValueMap()
	ns.Assign("a", types.FuString("foo"))
	ns.Assign("b", types.FuString("bar"))
	expect := types.FuString("foobar")
	assertEvaluateOK(t, ns, expect, addnode)

	// case 2: adding a function to a string fails
	ns.Assign("b", types.NewFixedFunction("b", 0, nil))
	assertEvaluateFail(t, ns,
		"loc1loc2: unsupported operation: cannot add function to string",
		addnode)

	// case 3: undefined name
	delete(ns, "b")
	assertEvaluateFail(t, ns, "loc2: name not defined: 'b'", addnode)
}

func stringnode(value string) *dsl.ASTString {
	// NewASTString takes a token, which comes quoted
	value = "\"" + value + "\""
	return dsl.NewASTString(value)
}

func assertIn(
	t *testing.T, ns types.ValueMap, name string, expect types.FuObject) {
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
	ns types.Namespace,
	expect types.FuObject,
	input dsl.ASTExpression) {

	obj, err := evaluate(ns, input)
	assert.Nil(t, err)

	if !expect.Equal(obj) {
		t.Errorf("expected\n%#v\nbut got\n%#v", expect, obj)
	}
}

func assertEvaluateFail(
	t *testing.T,
	ns types.Namespace,
	expecterr string,
	input dsl.ASTExpression) {

	obj, errs := evaluate(ns, input)
	assert.Equal(t, expecterr, errs[0].Error())
	assert.Nil(t, obj)
}
