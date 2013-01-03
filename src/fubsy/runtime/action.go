// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	//"fmt"
	"os"
	"os/exec"
	"strings"

	"fubsy/dsl"
	"fubsy/log"
	"fubsy/types"
)

type Action interface {
	String() string

	// Perform whatever task this action implies. Return nil on
	// success, error otherwise. Compound actions always fail on the
	// first error; they do not continue executing. (The global
	// "--keep-going" option is irrelevant at this level; the caller
	// of Execute() is responsible for respecting that option.)
	Execute(ns types.Namespace) []error
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

func (self *SequenceAction) Execute(ns types.Namespace) []error {
	for _, sub := range self.subactions {
		errs := sub.Execute(ns)
		if len(errs) > 0 {
			return errs
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

func (self *CommandAction) Execute(ns types.Namespace) []error {
	//fmt.Println(self.raw)

	var err error
	self.expanded, err = self.raw.Expand(ns)
	if err != nil {
		return []error{err}
	}
	log.Info("%s", self.expanded)

	// Run commands with the shell because people expect redirection,
	// pipes, etc. to work from their build scripts. (And besides, all
	// we have is a string: Fubsy makes no effort to encourage
	// commands as lists. That just confuses people and causes excess
	// typing. And it's pointless on Windows, where command lists get
	// collapsed to a string and then parsed back into words by the
	// program being run.)
	// XXX can we mitigate security risks of using the shell?
	// XXX what about Windows?
	// XXX for parallel builds: gather stdout and stderr, accumulate
	// them in order but still distinguishing them, and dump them to
	// our stdout/stderr when the command finishes
	// XXX the error message doesn't say which command failed (and if
	// it did, it would probably say "/bin/sh", which is useless): can
	// we do better?
	cmd := exec.Command("/bin/sh", "-c", self.expanded.String())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return []error{err}
	}
	return nil
}

func (self *AssignmentAction) String() string {
	return self.assignment.Target() + " = ..."
	//return self.assignment.String()
}

func (self *AssignmentAction) Execute(ns types.Namespace) []error {
	return assign(ns, self.assignment)
}

func (self *FunctionCallAction) String() string {
	return self.fcall.String() + "(...)"
}

func (self *FunctionCallAction) Execute(ns types.Namespace) []error {
	_, errs := evaluateCall(ns, self.fcall)
	return errs
}
