// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dsl"
)

func Test_SequenceAction_create(t *testing.T) {
	rt := &Runtime{}
	action := NewSequenceAction()
	assert.Equal(t, 0, len(action.subactions))

	// Execute() on an empty SequenceAction does nothing, silently
	assert.Nil(t, action.Execute(rt))

	// action 1 is a bare string: "ls -lR foo/bar"
	cmd := dsl.NewASTString("\"ls -lR foo/bar\"")
	action.AddCommand(cmd)

	// action 2: a = "foo"
	assign := dsl.NewASTAssignment(
		"a",
		dsl.NewASTString("foo"))
	action.AddAssignment(assign)

	// action 3: remove("*.o")
	fcall := dsl.NewASTFunctionCall(
		dsl.NewASTString("remove"),
		[]dsl.ASTExpression{dsl.NewASTString("\"*.c\"")})
	action.AddFunctionCall(fcall)

	assert.Equal(t, 3, len(action.subactions))
	assert.Equal(t,
		"ls -lR foo/bar",
		action.subactions[0].(*CommandAction).raw.ValueString())
}
