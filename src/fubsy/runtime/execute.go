// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"fmt"

	"fubsy/dag"
	"fubsy/dsl"
	"fubsy/types"
)

// Execute Fubsy code by inspecting AST nodes and updating namespaces,
// the DAG, etc. This is the workhorse backend of both Runtime and
// BuildRule. Runtime uses it for executing code in the main phase
// (and eventually other phases), and BuildRule uses it for executing
// actions in the build phase.

// node represents code like "NAME = EXPR": evaluate EXPR and store
// the result in self's namespace
func (self *Runtime) assign(node *dsl.ASTAssignment) []error {
	value, err := self.evaluate(node.Expression())
	if err != nil {
		return err
	}
	self.stack.Assign(node.Target(), value)
	return nil
}

func (self *Runtime) evaluate(
	expr_ dsl.ASTExpression) (
	result types.FuObject, errs []error) {
	switch expr := expr_.(type) {
	case *dsl.ASTString:
		result = types.MakeFuString(expr.Value())
	case *dsl.ASTList:
		result, errs = self.evaluateList(expr)
	case *dsl.ASTName:
		result, errs = self.evaluateName(expr)
	case *dsl.ASTFileFinder:
		result = dag.NewFinderNode(expr.Patterns()...)
	case *dsl.ASTAdd:
		result, errs = self.evaluateAdd(expr)
	case *dsl.ASTFunctionCall:
		var callable types.FuCallable
		var args RuntimeArgs
		callable, args, errs = self.prepareCall(expr)
		if len(errs) == 0 {
			result, errs = self.evaluateCall(callable, args, nil)
		}
	case *dsl.ASTSelection:
		_, result, errs = self.evaluateLookup(expr)
	default:
		return nil, []error{unsupportedAST(expr_)}
	}
	for i, err := range errs {
		errs[i] = MakeLocationError(expr_, err)
	}
	return
}

func (self *Runtime) evaluateList(expr *dsl.ASTList) (types.FuObject, []error) {
	elements := expr.Elements()
	values := make([]types.FuObject, len(elements))
	var allerrs []error
	var errs []error
	for i, element := range elements {
		values[i], errs = self.evaluate(element)
		if len(errs) != 0 {
			allerrs = append(allerrs, errs...)
		}
	}
	if allerrs != nil {
		return nil, allerrs
	}
	return types.MakeFuList(values...), nil
}

func (self *Runtime) evaluateName(expr *dsl.ASTName) (types.FuObject, []error) {
	name := expr.Name()
	value, ok := self.Lookup(name)
	if !ok {
		err := fmt.Errorf("name not defined: '%s'", name)
		return nil, []error{err}
	}
	return value, nil
}

func (self *Runtime) evaluateAdd(expr *dsl.ASTAdd) (types.FuObject, []error) {
	op1, op2 := expr.Operands()
	obj1, errs := self.evaluate(op1)
	if len(errs) > 0 {
		return nil, errs
	}
	obj2, errs := self.evaluate(op2)
	if len(errs) > 0 {
		return nil, errs
	}
	result, err := obj1.Add(obj2)
	if err != nil {
		return nil, []error{err}
	}
	return result, nil
}

func (self *Runtime) prepareCall(expr *dsl.ASTFunctionCall) (
	callable types.FuCallable, args RuntimeArgs, errs []error) {

	// robj is the receiver object for a method call (foo in foo.x())
	// value is the callable object (function or method) as a FuObject
	var robj, value types.FuObject
	args.runtime = self

	// two cases to worry about here:
	//    1. fn(...)
	//    2. robj.meth(...)
	astfunc := expr.Function()
	if astselect, ok := astfunc.(*dsl.ASTSelection); ok {
		// case 2: looks like a method call; we need to keep track of
		// the receiver object
		robj, value, errs = self.evaluateLookup(astselect)
	} else {
		// case 1: it's a normal function call, so robj stays nil
		value, errs = self.evaluate(expr.Function())
	}
	if len(errs) > 0 {
		return
	}
	args.SetReceiver(robj)

	callable, ok := value.(types.FuCallable)
	if !ok {
		errs = []error{
			fmt.Errorf("not a function or method: '%s'", expr.Function())}
		return
	}

	var astargs []dsl.ASTExpression
	astargs = expr.Args()
	arglist := make([]types.FuObject, len(astargs))
	for i, astarg := range astargs {
		arglist[i], errs = self.evaluate(astarg)
		if len(errs) > 0 {
			return
		}
	}
	args.SetArgs(arglist)
	errs = nil
	return
}

func (self *Runtime) expandArgs(argsource RuntimeArgs) (RuntimeArgs, []error) {
	var errs []error
	var err error

	// XXX ignoring kwargs
	args := argsource.Args()
	xargs := make([]types.FuObject, len(args))
	for i, arg := range args {
		xargs[i], err = arg.ActionExpand(self.stack, nil)
		if err != nil {
			errs = append(errs, err)
		}
	}

	result := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(argsource.Receiver(), xargs, nil),
		runtime:   argsource.runtime,
	}
	return result, errs
}

func (self *Runtime) evaluateCall(
	callable types.FuCallable,
	args RuntimeArgs,
	precall func(types.FuCallable, types.ArgSource)) (
	types.FuObject, []error) {

	if precall != nil {
		precall(callable, args)
	}
	err := callable.CheckArgs(args)
	if err != nil {
		return nil, []error{err}
	}
	return callable.Code()(args)
}

func (self *Runtime) evaluateLookup(expr *dsl.ASTSelection) (
	container, value types.FuObject, errs []error) {

	container, errs = self.evaluate(expr.Container())
	if len(errs) > 0 {
		return
	}
	var ok bool
	value, ok = container.Lookup(expr.Name())
	if !ok {
		errs = append(errs,
			fmt.Errorf("%s %s has no attribute '%s'",
				container.Typename(), container, expr.Name()))
		return
	}
	return
}

type LocationError struct {
	location dsl.Location
	err      error
}

func MakeLocationError(loc dsl.Locatable, err error) error {
	if _, ok := err.(LocationError); ok {
		return err
	}
	return LocationError{loc.Location(), err}
}

func MakeLocationErrors(loc dsl.Locatable, errs []error) []error {
	if errs == nil || len(errs) == 0 {
		return errs
	}
	for i, err := range errs {
		errs[i] = MakeLocationError(loc, err)
	}
	return errs
}

func (self LocationError) Error() string {
	if self.location == nil {
		return self.err.Error()
	}
	return self.location.ErrorPrefix() + self.err.Error()
}

type RuntimeArgs struct {
	types.BasicArgs
	runtime *Runtime
}

// other methods that might come in handy
func (self RuntimeArgs) Graph() *dag.DAG {
	return self.runtime.dag
}
