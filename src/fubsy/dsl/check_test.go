package dsl

import (
	"testing"
	"reflect"
)

func Test_checkActions_ok(t *testing.T) {
	nodes := []ASTNode {
		ASTString{value: "foo"},
		ASTFunctionCall{function: ASTName{name: "foo"}},
		ASTAssignment{target: "x", expr: ASTFunctionCall{}},
		ASTFunctionCall{
			function: ASTName{name: "blah"},
			args: []ASTExpression {ASTString{}, ASTString{}}},
	}
	actions, errors := checkActions(nodes)
	assertTrue(t, reflect.DeepEqual(actions, nodes),
		"expected %d valid actions, but got %d: %v",
		len(nodes), len(actions), actions)
	assertTrue(t, len(errors) == 0,
		"expected no errors, but got %d: %v", len(errors), errors)
}

func Test_checkActions_bad(t *testing.T) {
	nodes := []ASTNode {
		ASTString{value: "foo bar"},	  // good
		ASTFileList{patterns: []string {"*.java"}}, // bad
		ASTFunctionCall{},				  // good
		ASTBuildRule{					  // bad
			targets: ASTString{value: "target"},
			sources: ASTString{value: "source"},
			actions: []ASTNode {},
		},
		// hmmm: lots of expressions evaluate to a string -- why can't
		// I say cmd = "cc -o foo foo.c" outside a build rule, and then
		// reference cmd inside the build rule?
		ASTName{name: "blah"},	// bad (currently)
	}
	expect_actions := []ASTNode {
		nodes[0],
		nodes[2],
	}
	expect_errors := []SemanticError {
		SemanticError{node: nodes[1]},
		SemanticError{node: nodes[3]},
		SemanticError{node: nodes[4]},
	}
	actions, errors := checkActions(nodes)
	assertTrue(t, len(expect_errors) == len(errors),
		"expected %d errors, but got %d: %v",
		len(expect_errors), len(errors), errors)
	for i, err := range expect_errors {
		enode := err.node
		anode := errors[i].(SemanticError).node
		assertTrue(t, anode.Equal(enode),
			"bad node %d: expected\n%T %v\nbut got\n%T %v",
			i, enode, enode, anode, anode)
	}
	assertTrue(t, reflect.DeepEqual(expect_actions, actions),
		"expected actions:\n%#v\nbut got:\n%#v",
		expect_actions, actions)
}
