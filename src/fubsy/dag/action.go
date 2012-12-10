package dag

import (
	"fmt"
	"os"
	"strings"
	"fubsy/dsl"
	"fubsy/types"
)

type Action interface {
	String() string

	// Perform whatever task this action implies. Return nil on
	// success, error otherwise. Compound actions always fail on the
	// first error; they do not continue executing. (The global
	// "--keep-going" option is irrelevant at this level; the caller
	// of Execute() is responsible for respecting that option.)
	Execute(ns types.Namespace) error
}

type actionbase struct {
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
	raw types.FuObject

	// with all variables expanded, ready to execute
	expanded types.FuObject
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

func NewSequenceAction() *SequenceAction {
	result := new(SequenceAction)
	return result
}

func (self *SequenceAction) String() string {
	result := make([]string, len(self.subactions))
	for i, sub := range self.subactions {
		result[i] = sub.String()
	}
	var tail string
	if len(result) > 3 {
		result = result[0:3]
		tail = " && ..."
	}
	return strings.Join(result, " && ") + tail
}

func (self *SequenceAction) Execute(ns types.Namespace) error {
	var err error
	for _, sub := range self.subactions {
		err = sub.Execute(ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SequenceAction) AddAction(action Action) {
	self.subactions = append(self.subactions, action)
}

func (self *SequenceAction) AddCommand(command *dsl.ASTString) {
	raw := types.FuString(command.Value())
	self.AddAction(&CommandAction{raw: raw})
}

func (self *SequenceAction) AddAssignment(assignment *dsl.ASTAssignment) {
	self.AddAction(&AssignmentAction{assignment: assignment})
}

func (self *SequenceAction) AddFunctionCall(fcall *dsl.ASTFunctionCall) {
	self.AddAction(&FunctionCallAction{fcall: fcall})
}


func (self *CommandAction) String() string {
	return self.raw.String()
}

func (self *CommandAction) Execute(ns types.Namespace) error {
	fmt.Println("raw command:", self.raw)
	fmt.Println("namespace for expansion:")
	ns.Dump(os.Stdout, "")

	command, err := self.raw.Expand(ns)
	if err != nil {
		return err
	}
	fmt.Println("command:", command)
	panic("command execution not implemented yet")

	// self.expanded = self.rule.Expand(self.raw)
	// fmt.Printf("execute:", self.expanded)
}

func (self *AssignmentAction) String() string {
	return self.assignment.Target() + " = ..."
	//return self.assignment.String()
}

func (self *AssignmentAction) Execute(ns types.Namespace) error {
	panic("assignment in build rule not implemented yet")
}

func (self *FunctionCallAction) String() string {
	return self.fcall.String() + "(...)"
}

func (self *FunctionCallAction) Execute(ns types.Namespace) error {
	panic("function call in build rule not implemeneted yet")
}
