package runtime

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"fubsy/dsl"
)

func Test_SequenceAction_create(t *testing.T) {
	runtime := NewRuntime("", nil)
	rule := NewBuildRule(runtime, nil, nil)
	action := NewSequenceAction(rule)
	assert.Equal(t, rule, action.rule)
	assert.Equal(t, 0, len(action.subactions))

	// Execute() on an empty SequenceAction does nothing, silently
	assert.Nil(t, action.Execute())

	// action 1 is a bare string: "ls -lR foo/bar"
	cmd := dsl.NewASTString("\"ls -lR foo/bar\"")
	action.addCommand(cmd)

	// action 2: a = "foo"
	assign := dsl.NewASTAssignment(
		"a",
		dsl.NewASTString("foo"))
	action.addAssignment(assign)

	// action 3: remove("*.o")
	fcall := dsl.NewASTFunctionCall(
		dsl.NewASTString("remove"),
		[]dsl.ASTExpression {dsl.NewASTString("\"*.c\"")})
	action.addFunctionCall(fcall)

	assert.Equal(t, 3, len(action.subactions))
	assert.Equal(t, "ls -lR foo/bar", action.subactions[0].(*CommandAction).raw)
}
