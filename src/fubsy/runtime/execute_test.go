// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/types"
)

func Test_assign(t *testing.T) {
	// AST for a = "foo"
	node := dsl.NewASTAssignment("a", stringnode("foo"))
	rt := minimalRuntime()

	errs := rt.assign(node)
	assert.Equal(t, 0, len(errs))
	expect := types.FuString("foo")
	assertIn(t, rt.Namespace(), "a", expect)

	// AST for a = foo (another variable, to provoke an error)
	node = dsl.NewASTAssignment("b", dsl.NewASTName("foo"))
	errs = rt.assign(node)
	assert.Equal(t, "name not defined: 'foo'", errs[0].Error())
	_, ok := rt.Lookup("b")
	assert.False(t, ok)
}

// evaluate simple expressions (no operators)
func Test_evaluate_simple(t *testing.T) {
	// the expression "meep" evaluates to the string "meep"
	var expect types.FuObject
	snode := stringnode("meep")
	rt := minimalRuntime()
	ns := rt.Namespace()
	expect = types.FuString("meep")
	assertEvaluateOK(t, rt, expect, snode)

	// the expression foo evaluates to the string "meep" if foo is set
	// to that string
	ns.Assign("foo", expect)
	nnode := dsl.NewASTName("foo")
	assertEvaluateOK(t, rt, expect, nnode)

	// ... and to an error if the variable is not defined
	location := dsl.NewStubLocation("hello, sailor")
	nnode = dsl.NewASTName("boo", location)
	assertEvaluateFail(t, rt, "hello, sailor: name not defined: 'boo'", nnode)

	// expression <*.c blah> evaluates to a FinderNode with two
	// include patterns
	patterns := []string{"*.c", "blah"}
	fnode := dsl.NewASTFileFinder(patterns)
	expect = dag.NewFinderNode("*.c", "blah")
	assertEvaluateOK(t, rt, expect, fnode)
}

// evaluate more complex expressions
func Test_evaluate_complex(t *testing.T) {
	// a + b evaluates to various things, depending on the value
	// of those two variables
	addnode := dsl.NewASTAdd(
		dsl.NewASTName("a", dsl.NewStubLocation("loc1")),
		dsl.NewASTName("b", dsl.NewStubLocation("loc2")))

	// case 1: two strings just get concatenated
	rt := minimalRuntime()
	ns := rt.Namespace()
	ns.Assign("a", types.FuString("foo"))
	ns.Assign("b", types.FuString("bar"))
	expect := types.FuString("foobar")
	assertEvaluateOK(t, rt, expect, addnode)

	// case 2: adding a function to a string fails
	ns.Assign("b", types.NewFixedFunction("b", 0, nil))
	assertEvaluateFail(t, rt,
		"loc1loc2: unsupported operation: cannot add function to string",
		addnode)

	// case 3: undefined name
	delete((*ns.(*types.ValueStack))[0].(types.ValueMap), "b")
	assertEvaluateFail(t, rt, "loc2: name not defined: 'b'", addnode)
}

func Test_evaluateCall(t *testing.T) {
	// foo() takes no args and always succeeds;
	// bar() takes exactly one arg and always fails
	calls := make([]string, 0) // list of function names

	fn_foo := func(args types.ArgSource) (types.FuObject, []error) {
		if len(args.Args()) != 0 {
			panic("foo() called with wrong number of args")
		}
		calls = append(calls, "foo")
		return types.FuString("foo!"), nil
	}
	fn_bar := func(args types.ArgSource) (types.FuObject, []error) {
		if len(args.Args()) != 1 {
			panic("bar() called with wrong number of args")
		}
		calls = append(calls, "bar")
		return nil, []error{
			fmt.Errorf("bar failed (%s)", args.Arg(0))}
	}

	rt := minimalRuntime()
	ns := rt.Namespace()
	ns.Assign("foo", types.NewFixedFunction("foo", 0, fn_foo))
	ns.Assign("bar", types.NewFixedFunction("bar", 1, fn_bar))
	ns.Assign("src", types.FuString("main.c"))

	var result types.FuObject
	var errors []error

	fooname := dsl.NewASTName("foo")
	barname := dsl.NewASTName("bar")
	noargs := []dsl.ASTExpression{}
	onearg := []dsl.ASTExpression{dsl.NewASTString("\"meep\"")}
	exparg := []dsl.ASTExpression{dsl.NewASTString("\">$src<\"")}

	// call foo() correctly (no args)
	ast := dsl.NewASTFunctionCall(fooname, noargs)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Equal(t, types.FuString("foo!"), result)
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, []string{"foo"}, calls)

	// call foo() incorrectly (1 arg)
	ast = dsl.NewASTFunctionCall(fooname, onearg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"function foo() takes no arguments (got 1)", errors[0].Error())
	assert.Equal(t, []string{"foo"}, calls)

	// call bar() correctly (1 arg)
	ast = dsl.NewASTFunctionCall(barname, onearg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "bar failed (\"meep\")", errors[0].Error())
	assert.Equal(t, []string{"foo", "bar"}, calls)

	// call bar() with an arg that needs to be expanded
	ast = dsl.NewASTFunctionCall(barname, exparg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "bar failed (\">main.c<\")", errors[0].Error())
	assert.Equal(t, []string{"foo", "bar", "bar"}, calls)

	// again, but this time expansion fails (undefined name)
	exparg = []dsl.ASTExpression{dsl.NewASTString("\"a $bogus value\"")}
	ast = dsl.NewASTFunctionCall(barname, exparg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t, "undefined variable 'bogus' in string", errors[0].Error())
	assert.Equal(t, []string{"foo", "bar", "bar"}, calls)

	// call bar() incorrectly (no args)
	ast = dsl.NewASTFunctionCall(barname, noargs)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"function bar() takes exactly 1 arguments (got 0)", errors[0].Error())
	assert.Equal(t, []string{"foo", "bar", "bar"}, calls)

	// call bar() incorrectly (1 arg, but it's an undefined name)
	ast = dsl.NewASTFunctionCall(
		barname, []dsl.ASTExpression{dsl.NewASTName("bogus")})
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"name not defined: 'bogus'", errors[0].Error())

	// attempt to call non-existent function
	ast = dsl.NewASTFunctionCall(dsl.NewASTName("bogus"), onearg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"name not defined: 'bogus'", errors[0].Error())

	// attempt to call something that is not a function
	ast = dsl.NewASTFunctionCall(dsl.NewASTName("src"), onearg)
	result, errors = rt.evaluateCall(ast, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errors))
	assert.Equal(t,
		"not a function or method: 'src'", errors[0].Error())

	assert.Equal(t, []string{"foo", "bar", "bar"}, calls)
}

func Test_evaluateCall_method(t *testing.T) {
	// construct AST for "a.b.c(x)"
	args := []dsl.ASTExpression{dsl.NewASTName("x")}
	ast := dsl.NewASTFunctionCall(
		dsl.NewASTSelection(
			dsl.NewASTSelection(dsl.NewASTName("a"), "b"), "c"),
		args)

	// make sure a is an object with attributes, and b is one of them
	// (N.B. having FileNodes be attributes of one another is weird
	// and would never happen in a real Fubsy script, but it's a
	// convenient way to setup this method call)
	aobj := dag.NewFileNode("a.txt")
	bobj := dag.NewFileNode("b.txt")
	aobj.ValueMap = types.NewValueMap()
	aobj.Assign("b", bobj)

	// make sure a.b.c is a method
	calls := make([]string, 0) // list of function names
	var meth_c types.FuCode
	meth_c = func(args types.ArgSource) (types.FuObject, []error) {
		if len(args.Args()) != 1 {
			panic("c() called with wrong number of args")
		}
		calls = append(calls, "c")
		robj := args.Receiver()
		return nil, []error{
			fmt.Errorf("c failed: receiver: %s %v, arg: %s %v",
				robj.Typename(), robj, args.Arg(0).Typename(), args.Arg(0))}
	}
	bobj.ValueMap = types.NewValueMap()
	bobj.Assign("c", types.NewFixedFunction("c", 1, meth_c))

	rt := minimalRuntime()
	ns := rt.Namespace()
	ns.Assign("a", aobj)
	ns.Assign("x", types.FuString("hello"))

	// what the hell, let's test the precall feature too
	var precalledExpr dsl.ASTExpression
	var precalledArgs types.FuObject
	precall := func(expr *dsl.ASTFunctionCall, args types.FuList) {
		precalledExpr = expr
		precalledArgs = args
	}

	result, errs := rt.evaluateCall(ast, precall)
	assert.Equal(t, precalledExpr, ast)
	assert.Equal(t, precalledArgs, types.MakeFuList("hello"))
	assert.Nil(t, result)
	if len(errs) == 1 {
		assert.Equal(t, "c failed: receiver: FileNode b.txt, arg: string \"hello\"", errs[0].Error())
	} else {
		t.Errorf("expected exactly 1 error, but got: %v", errs)
	}
}

func Test_LocationError(t *testing.T) {
	var loc dsl.Locatable
	loc = dsl.NewStubLocation("right here")
	err := errors.New("it hurts!")
	locerr := MakeLocationError(loc, err)
	assert.Equal(t, "right here: it hurts!", locerr.Error())

	// make sure it still works when LocationError has a Locatable
	// that wraps the real Location
	loc = dsl.NewStubLocatable(loc.(dsl.Location))
	locerr = MakeLocationError(loc, err)
	assert.Equal(t, "right here: it hurts!", locerr.Error())

	// and finally, don't crash when LocationError has a Locatable
	// that wraps a nil Location
	loc = dsl.NewStubLocatable(nil)
	locerr = MakeLocationError(loc, err)
	assert.Equal(t, "it hurts!", locerr.Error())
}

func stringnode(value string) *dsl.ASTString {
	// NewASTString takes a token, which comes quoted
	value = "\"" + value + "\""
	return dsl.NewASTString(value)
}

func assertIn(
	t *testing.T, ns types.Namespace, name string, expect types.FuObject) {
	if actual, ok := ns.Lookup(name); ok {
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

	if !expect.Equal(obj) {
		t.Errorf("expected\n%#v\nbut got\n%#v", expect, obj)
	}
}

func assertEvaluateFail(
	t *testing.T,
	rt *Runtime,
	expecterr string,
	input dsl.ASTExpression) {

	obj, errs := rt.evaluate(input)
	assert.Equal(t, expecterr, errs[0].Error())
	assert.Nil(t, obj)
}
