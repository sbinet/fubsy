// Copyright © 2012, Greg Ward. All rights reserved.
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
// the result in ns
func assign(ns types.Namespace, node *dsl.ASTAssignment) []error {
	value, err := evaluate(ns, node.Expression())
	if err != nil {
		return err
	}
	ns.Assign(node.Target(), value)
	return nil
}

func evaluate(
	ns types.Namespace, expr_ dsl.ASTExpression) (types.FuObject, []error) {
	switch expr := expr_.(type) {
	case *dsl.ASTString:
		return types.FuString(expr.Value()), nil
	case *dsl.ASTName:
		return evaluateName(ns, expr)
	case *dsl.ASTFileFinder:
		return dag.NewFinderNode(expr.Patterns()), nil
	case *dsl.ASTAdd:
		return evaluateAdd(ns, expr)
	case *dsl.ASTFunctionCall:
		return evaluateCall(ns, expr)
	default:
		return nil, []error{unsupportedAST(expr_)}
	}
	panic("unreachable code")
}

func evaluateName(
	ns types.Namespace, expr *dsl.ASTName) (types.FuObject, []error) {
	name := expr.Name()
	value, ok := ns.Lookup(name)
	if !ok {
		err := RuntimeError{
			location: expr.Location(),
			message:  fmt.Sprintf("name not defined: '%s'", name),
		}
		return nil, []error{err}
	}
	return value, nil
}

func evaluateAdd(
	ns types.Namespace, expr *dsl.ASTAdd) (types.FuObject, []error) {
	op1, op2 := expr.Operands()
	obj1, errs := evaluate(ns, op1)
	if len(errs) > 0 {
		return nil, errs
	}
	obj2, errs := evaluate(ns, op2)
	if len(errs) > 0 {
		return nil, errs
	}
	result, err := obj1.Add(obj2)
	if err != nil {
		return nil, []error{err}
	}
	return result, nil
}

func evaluateCall(
	ns types.Namespace, expr *dsl.ASTFunctionCall) (types.FuObject, []error) {
	value, errs := evaluate(ns, expr.Function())
	if len(errs) > 0 {
		return nil, errs
	}
	function, ok := value.(types.FuCallable)
	if !ok {
		err := RuntimeError{
			expr.Location(),
			fmt.Sprintf("not a function or method: '%s'", expr.Function()),
		}
		return nil, []error{err}
	}
	astargs := expr.Args() // slice of ASTExpression
	args := make(types.FuList, len(astargs))
	for i, astarg := range astargs {
		args[i], errs = evaluate(ns, astarg)
		if len(errs) > 0 {
			return nil, errs
		}
	}

	fmt.Printf("function = %v, args = %v\n", function, args)
	err := function.CheckArgs(args)
	if err != nil {
		return nil, []error{err}
	}
	return function.Code()(args, nil)
}
