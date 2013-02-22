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

func Test_prepareCall(t *testing.T) {
	// this is never going to be called, so it's OK that it's nil
	var fn_dummy func(argsource types.ArgSource) (types.FuObject, []error)
	var dummy1, dummy2 types.FuCallable
	dummy1 = types.NewFixedFunction("dummy1", 0, fn_dummy)
	dummy2 = types.NewFixedFunction("dummy1", 1, fn_dummy)

	rt := minimalRuntime()
	ns := rt.Namespace()
	ns.Assign("dummy1", dummy1)
	ns.Assign("dummy2", dummy2)
	ns.Assign("x", types.FuString("whee!"))

	noargs := []dsl.ASTExpression{}
	onearg := []dsl.ASTExpression{dsl.NewASTString("\"meep\"")}

	var astcall *dsl.ASTFunctionCall
	var callable types.FuCallable
	var args RuntimeArgs
	var errs []error

	// correct (no args) call to dummy1()
	astcall = dsl.NewASTFunctionCall(dsl.NewASTName("dummy1"), noargs)
	callable, args, errs = rt.prepareCall(astcall)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, dummy1, callable)
	assert.Equal(t, []types.FuObject{}, args.Args())

	// and to dummy2()
	astcall = dsl.NewASTFunctionCall(dsl.NewASTName("dummy2"), onearg)
	callable, args, errs = rt.prepareCall(astcall)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, dummy2, callable)
	assert.Equal(t, []types.FuObject{types.FuString("meep")}, args.Args())

	// attempt to call dummy2() incorrectly (1 arg, but it's an undefined name)
	astcall = dsl.NewASTFunctionCall(
		dsl.NewASTName("dummy2"),
		[]dsl.ASTExpression{dsl.NewASTName("bogus")})
	callable, _, errs = rt.prepareCall(astcall)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "name not defined: 'bogus'", errs[0].Error())

	// attempt to call non-existent function
	astcall = dsl.NewASTFunctionCall(dsl.NewASTName("bogus"), noargs)
	callable, _, errs = rt.prepareCall(astcall)
	assert.Nil(t, callable)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "name not defined: 'bogus'", errs[0].Error())

	// attempt to call something that is not a function
	astcall = dsl.NewASTFunctionCall(dsl.NewASTName("x"), noargs)
	callable, _, errs = rt.prepareCall(astcall)
	assert.Nil(t, callable)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "not a function or method: 'x'", errs[0].Error())
}

func Test_evaluateCall(t *testing.T) {
	// foo() takes no args and always succeeds;
	// bar() takes exactly one arg and always fails
	calls := make([]string, 0) // list of function names

	fn_foo := func(argsource types.ArgSource) (types.FuObject, []error) {
		if len(argsource.Args()) != 0 {
			panic("foo() called with wrong number of args")
		}
		calls = append(calls, "foo")
		return types.FuString("foo!"), nil
	}
	fn_bar := func(argsource types.ArgSource) (types.FuObject, []error) {
		if len(argsource.Args()) != 1 {
			panic("bar() called with wrong number of args")
		}
		calls = append(calls, "bar")
		return nil, []error{
			fmt.Errorf("bar failed (%s)", argsource.Args()[0])}
	}
	var foo, bar types.FuCallable
	foo = types.NewFixedFunction("foo", 0, fn_foo)
	bar = types.NewFixedFunction("bar", 1, fn_bar)

	rt := minimalRuntime()
	args := RuntimeArgs{runtime: rt}

	var result types.FuObject
	var errs []error

	// call foo() correctly (no args)
	args.SetArgs([]types.FuObject{})
	result, errs = rt.evaluateCall(foo, args, nil)
	assert.Equal(t, types.FuString("foo!"), result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo"}, calls)

	// call foo() incorrectly (1 arg)
	args.SetArgs([]types.FuObject{types.FuString("meep")})
	result, errs = rt.evaluateCall(foo, args, nil)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t,
		"function foo() takes no arguments (got 1)", errs[0].Error())
	assert.Equal(t, []string{"foo"}, calls)

	// call bar() correctly (1 arg)
	result, errs = rt.evaluateCall(bar, args, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "bar failed (\"meep\")", errs[0].Error())
	assert.Equal(t, []string{"foo", "bar"}, calls)

	// call bar() incorrectly (no args)
	args.SetArgs(nil)
	result, errs = rt.evaluateCall(bar, args, nil)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t,
		"function bar() takes exactly 1 arguments (got 0)", errs[0].Error())

	// check the sequence of calls
	assert.Equal(t, []string{"foo", "bar"}, calls)
}

func Test_evaluateCall_no_expand(t *testing.T) {
	calls := 0
	fn_foo := func(argsource types.ArgSource) (types.FuObject, []error) {
		calls++
		return types.FuString("arg: " + argsource.Args()[0].ValueString()), nil
	}
	foo := types.NewFixedFunction("foo", 1, fn_foo)
	rt := minimalRuntime()
	args := RuntimeArgs{runtime: rt}

	// call bar() with an arg that needs to be expanded to test that
	// expansion does *not* happen -- evaluateCall() doesn't know
	// which phase it's in, so it has to rely on someone else to
	// ActionExpand() each value in the build phase
	args.SetArgs([]types.FuObject{types.FuString(">$src<")})
	result, errs := rt.evaluateCall(foo, args, nil)
	assert.Equal(t, 1, calls)
	assert.Equal(t, types.FuString("arg: >$src<"), result)
	if len(errs) != 0 {
		t.Errorf("expected no errors, but got: %v", errs)
	}

	// now make a value that expands to three values
	expansion := types.MakeFuList("a", "b", "c")
	var val types.FuObject = types.NewStubObject("val", expansion)
	valexp, _ := val.ActionExpand(nil, nil)
	assert.Equal(t, expansion, valexp) // this actually tests StubObject

	// call foo() with that expandable value, and make sure it is
	// really called with the unexpanded value
	args.SetArgs([]types.FuObject{val})
	result, errs = rt.evaluateCall(foo, args, nil)
	assert.Equal(t, 2, calls)
	assert.Equal(t, types.FuString("arg: val"), result)
	if len(errs) != 0 {
		t.Errorf("expected no errors, but got: %v", errs)
	}
}

func Test_evaluateCall_method(t *testing.T) {
	// construct AST for "a.b.c(x)"
	astargs := []dsl.ASTExpression{dsl.NewASTName("x")}
	astcall := dsl.NewASTFunctionCall(
		dsl.NewASTSelection(
			dsl.NewASTSelection(dsl.NewASTName("a"), "b"), "c"),
		astargs)

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
	meth_c = func(argsource types.ArgSource) (types.FuObject, []error) {
		args := argsource.Args()
		if len(args) != 1 {
			panic("c() called with wrong number of args")
		}
		calls = append(calls, "c")
		robj := argsource.Receiver()
		return nil, []error{
			fmt.Errorf("c failed: receiver: %s %v, arg: %s %v",
				robj.Typename(), robj, args[0].Typename(), args[0])}
	}
	bobj.ValueMap = types.NewValueMap()
	bobj.Assign("c", types.NewFixedFunction("c", 1, meth_c))

	rt := minimalRuntime()
	ns := rt.Namespace()
	ns.Assign("a", aobj)
	ns.Assign("x", types.FuString("hello"))

	// what the hell, let's test the precall feature too
	var precalledCallable types.FuCallable
	var precalledArgs types.ArgSource
	precall := func(callable types.FuCallable, argsource types.ArgSource) {
		precalledCallable = callable
		precalledArgs = argsource
	}

	callable, args, errs := rt.prepareCall(astcall)
	assert.Equal(t, "c", callable.(*types.FuFunction).Name())
	assert.True(t, args.Receiver() == bobj)
	assert.Equal(t, 0, len(errs))

	result, errs := rt.evaluateCall(callable, args, precall)
	assert.Equal(t, precalledCallable, callable)
	assert.Equal(t,
		(types.FuList)(precalledArgs.Args()), types.MakeFuList("hello"))
	assert.Nil(t, result)
	if len(errs) == 1 {
		assert.Equal(t,
			"c failed: receiver: FileNode \"b.txt\", arg: string \"hello\"",
			errs[0].Error())
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
