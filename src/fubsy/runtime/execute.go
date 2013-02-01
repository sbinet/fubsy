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
		result = types.FuString(expr.Value())
	case *dsl.ASTName:
		result, errs = self.evaluateName(expr)
	case *dsl.ASTFileFinder:
		result = dag.NewFinderNode(expr.Patterns()...)
	case *dsl.ASTAdd:
		result, errs = self.evaluateAdd(expr)
	case *dsl.ASTFunctionCall:
		result, errs = self.evaluateCall(expr, nil)
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

func (self *Runtime) evaluateCall(
	expr *dsl.ASTFunctionCall,
	precall func(*dsl.ASTFunctionCall, types.FuList)) (
	types.FuObject, []error) {

	var robj, value types.FuObject
	var errs []error

	// two cases to worry about here:
	//    1. fn(...)
	//    2. robj.meth(...)
	astfunc := expr.Function()
	if astselect, ok := astfunc.(*dsl.ASTSelection); ok {
		// case 2: it's a method call; we need to keep track of the
		// receiver object
		robj, value, errs = self.evaluateLookup(astselect)
	} else {
		// case 1: it's a normal function call, so robj stays nil
		value, errs = self.evaluate(expr.Function())
	}
	if len(errs) > 0 {
		return nil, errs
	}

	var err error
	callable, ok := value.(types.FuCallable)
	if !ok {
		err = fmt.Errorf("not a function or method: '%s'", expr.Function())
		return nil, []error{err}
	}

	var astargs []dsl.ASTExpression
	var arglist types.FuList
	astargs = expr.Args()
	arglist = make(types.FuList, len(astargs))
	for i, astarg := range astargs {
		arglist[i], errs = self.evaluate(astarg)
		if len(errs) > 0 {
			return nil, errs
		}
	}

	if precall != nil {
		precall(expr, arglist)
	}

	err = callable.CheckArgs(arglist)
	if err != nil {
		return nil, []error{err}
	}
	args := FunctionArgs{
		runtime: self,
		robj:    robj,
		args:    arglist,
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

type FunctionArgs struct {
	runtime *Runtime
	robj    types.FuObject
	args    []types.FuObject
	kwargs  types.ValueMap
}

// implement types.ArgSource
func (self FunctionArgs) Receiver() types.FuObject {
	return self.robj
}

func (self FunctionArgs) Args() []types.FuObject {
	return self.args
}

func (self FunctionArgs) Arg(i int) types.FuObject {
	return self.args[i]
}

func (self FunctionArgs) KeywordArgs() types.ValueMap {
	return self.kwargs
}

func (self FunctionArgs) KeywordArg(name string) (types.FuObject, bool) {
	value, ok := self.kwargs[name]
	return value, ok
}

// other methods that might come in handy
func (self FunctionArgs) Graph() *dag.DAG {
	return self.runtime.dag
}
