// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

import (
	"github.com/stretchrcom/testify/assert"
	"reflect"
	"testing"
)

func Test_checkActions_ok(t *testing.T) {
	nodes := []ASTNode{
		&ASTString{value: "foo"},
		&ASTFunctionCall{function: &ASTName{name: "foo"}},
		&ASTAssignment{target: "x", expr: &ASTFunctionCall{}},
		&ASTFunctionCall{
			function: &ASTName{name: "blah"},
			args:     []ASTExpression{&ASTString{}, &ASTString{}}},
	}
	actions, errors := checkActions(nodes)
	assert.True(t, reflect.DeepEqual(actions, nodes),
		"expected %d valid actions, but got %d: %v",
		len(nodes), len(actions), actions)
	assert.Equal(t, 0, len(errors),
		"expected no errors")
}

func Test_checkActions_bad(t *testing.T) {
	// ensure that one of the bad nodes has location info so we can
	// test that SemanticError.Error() includes it
	fileinfo := &fileinfo{"foo.fubsy", []int{0, 10, 15, 16, 20}}
	location := Location{fileinfo, 11, 18} // line 2-4

	nodes := []ASTNode{
		&ASTString{value: "foo bar"},               // good
		&ASTFileList{patterns: []string{"*.java"}}, // bad
		&ASTFunctionCall{},                         // good
		&ASTBuildRule{ // bad
			astbase:  astbase{location},
			targets:  &ASTString{value: "target"},
			sources:  &ASTString{value: "source"},
			children: []ASTNode{},
		},
		// hmmm: lots of expressions evaluate to a string -- why can't
		// I say cmd = "cc -o foo foo.c" outside a build rule, and then
		// reference cmd inside the build rule?
		&ASTName{name: "blah"}, // bad (currently)
	}
	expect_actions := []ASTNode{
		nodes[0],
		nodes[2],
	}
	expect_errors := []SemanticError{
		SemanticError{node: nodes[1]},
		SemanticError{node: nodes[3]},
		SemanticError{node: nodes[4]},
	}
	actions, errors := checkActions(nodes)
	assert.True(t, len(expect_errors) == len(errors),
		"expected %d errors, but got %d: %v",
		len(expect_errors), len(errors), errors)
	for i, err := range expect_errors {
		enode := err.node
		anode := errors[i].(SemanticError).node
		assert.True(t, anode.Equal(enode),
			"bad node %d: expected\n%T %p\nbut got\n%T %p",
			i, enode, enode, anode, anode)
	}

	expect_message := "foo.fubsy:2-4: invalid build action: must be either bare string, function call, or variable assignment"
	actual_message := errors[1].Error()
	assert.Equal(t, expect_message, actual_message)

	assert.True(t, reflect.DeepEqual(expect_actions, actions),
		"expected actions:\n%#v\nbut got:\n%#v",
		expect_actions, actions)
}
