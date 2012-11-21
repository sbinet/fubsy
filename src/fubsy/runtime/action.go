package runtime

import (
	"fmt"
	"fubsy/dsl"
)

type Action interface {
	// Perform whatever task this action implies. Return nil on
	// success, error otherwise. Compound actions always fail on the
	// first error; they do not continue executing. (The global
	// "--keep-going" option is irrelevant this deep in the build
	// system; it's used higher up, walking the graph of dependencies
	// and executing the action of each stale target.)
	Execute() error
}

type actionbase struct {
	runtime *Runtime
	rule *BuildRule
}

// an action that just consists of a list of other actions
type SequenceAction struct {
	actionbase
	subactions []Action
}

// an action that is a shell command to execute
type CommandAction struct {
	actionbase

	// as read from the build script, without variables expanded
	raw string

	// with all variables expanded, ready to execute
	expanded string
}

// an action that evaluates an expression and assigns the result to a
// local variable -- only affects the scope of one build rule
type AssignmentAction struct {
	actionbase
	assignment *dsl.ASTAssignment
}

// an action that calls a function with real-world side effects (e.g.
// remove(), copyfile()) -- a pure function would be useless here,
// since we do nothing with the return value!
type FunctionCallAction struct {
	actionbase
	fcall *dsl.ASTFunctionCall
}

func NewSequenceAction(rule *BuildRule) *SequenceAction {
	result := new(SequenceAction)
	result.rule = rule
	return result
}

func (self *SequenceAction) Execute() error {
	var err error
	for _, sub := range self.subactions {
		err = sub.Execute()
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SequenceAction) addAction(action Action) {
	self.subactions = append(self.subactions, action)
}

func (self *SequenceAction) addCommand(command *dsl.ASTString) {
	self.addAction(&CommandAction{raw: command.Value()})
}

func (self *SequenceAction) addAssignment(assignment *dsl.ASTAssignment) {
	self.addAction(&AssignmentAction{assignment: assignment})
}

func (self *SequenceAction) addFunctionCall(fcall *dsl.ASTFunctionCall) {
	self.addAction(&FunctionCallAction{fcall: fcall})
}


func (self *CommandAction) Execute() error {
	fmt.Println("expand:", self.raw)
	panic("command execution not implemented yet")

	// self.expanded = self.rule.Expand(self.raw)
	// fmt.Printf("execute:", self.expanded)
}

func (self *AssignmentAction) Execute() error {
	panic("assignment in build rule not implemented yet")
}

func (self *FunctionCallAction) Execute() error {
	panic("function call in build rule not implemeneted yet")
}
