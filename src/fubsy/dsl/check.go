// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dsl

// post-parse AST verification (detect semantic errors)

type SemanticError struct {
	node    ASTNode
	message string
}

func (self SemanticError) Error() string {
	return self.node.Location().String() + self.message
}

func checkAST(ast *ASTRoot) []error {
	// ugh: need generic AST walker?
	errors := make([]error, 0)
	for _, elem_ := range ast.children {
		if elem, ok := elem_.(*ASTPhase); ok {
			for _, stmt_ := range elem.children {
				if stmt, ok := stmt_.(*ASTBuildRule); ok {
					actions, brerrors := checkActions(stmt.children)
					stmt.children = actions
					errors = append(errors, brerrors...)
				}
			}
		}
	}
	return errors
}

// Check if all of the statements in nodes are valid actions for a
// build rule: either a bare string (shell command), a function call,
// or a variable assignment. Return a list of valid action nodes
// and a list of error objects for the invalid ones.
func checkActions(nodes []ASTNode) (actions []ASTNode, errors []error) {
	actions = make([]ASTNode, 0, len(nodes))
	for _, node := range nodes {
		_, ok1 := node.(*ASTString)
		_, ok2 := node.(*ASTFunctionCall)
		_, ok3 := node.(*ASTAssignment)
		if !(ok1 || ok2 || ok3) {
			errors = append(errors, SemanticError{
				node:    node,
				message: "invalid build action: must be either bare string, function call, or variable assignment"})
		} else {
			actions = append(actions, node)
		}
	}
	return actions, errors
}
