// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"fmt"
)

// object passed to every Fubsy function or method that encapsulates
// all the information required by it
type ArgSource interface {
	Receiver() FuObject
	Args() []FuObject
	Arg(i int) FuObject
	KeywordArgs() ValueMap
	KeywordArg(name string) (FuObject, bool)
}

// the inner heart of a function or method, the code that is actually called
// XXX should we allow multiple return values ([]FuObject)?
type FuCode func(args ArgSource) (FuObject, []error)

// Every function (method) may take required (positional) arguments
// and optional (keyword) arguments. Each function specifies how many
// positional args it takes and which keyword args it takes. Fubsy's
// runtime is responsible for making sure that callers conform to the
// function's requirements, so functions don't have to do their own
// argument count checking.
//
// Some examples:
// 	 foo()               		 # exactly zero args
// 	 foo(a)              		 # exactly 1 arg
// 	 foo(...)            		 # any number of args ("at least zero")
// 	 foo(a, b, c, ...)   		 # at least 3 args
// 	 foo(a, b, x=c, y=c, z=c)    # exactly 2 plus kwargs
// 	 foo(a, b, c, x=c, y=c)      # at least 2 plus kwargs
//
// XXX support for keywords not implemented in the parser, so it's
// not reflected in CheckArgs() either
type FuCallable interface {
	FuObject
	Name() string
	Code() FuCode

	// check that the arguments being passed are valid for this function,
	// returning a user-targeted error object if not
	CheckArgs(args []FuObject) error
}

type FuFunction struct {
	name    string
	minargs int
	maxargs int
	optargs []string
	code    FuCode
}

func NewFixedFunction(name string, numargs int, code FuCode) *FuFunction {
	return &FuFunction{name, numargs, numargs, nil, code}
}

func NewVariadicFunction(name string, minargs, maxargs int, code FuCode) *FuFunction {
	return &FuFunction{name, minargs, maxargs, nil, code}
}

func (self *FuFunction) String() string {
	return self.name + "()"
}

func (self *FuFunction) CommandString() string {
	// hmmm: perhaps CommandString needs an error return...
	panic("functions should not be interpolated into command strings!")
}

func (self *FuFunction) Equal(other_ FuObject) bool {
	other, ok := other_.(*FuFunction)
	return ok && &self.code == &other.code && self.name == other.name
}

func (self *FuFunction) Add(other_ FuObject) (FuObject, error) {
	return nil, unsupportedOperation(self, other_, "cannot add %s to %s")
}

func (self *FuFunction) Lookup(name string) (FuObject, bool) {
	return DefaultLookup(self, name)
}

func (self *FuFunction) List() []FuObject {
	return []FuObject{self}
}

func (self *FuFunction) ActionExpand(ns Namespace, ctx *ExpandContext) (FuObject, error) {
	return self, nil
}

func (self *FuFunction) Typename() string {
	return "function"
}

func (self *FuFunction) Name() string {
	return self.name
}

func (self *FuFunction) SetOptionalArgs(arg ...string) {
	self.optargs = arg
}

func (self *FuFunction) SetCode(code FuCode) {
	self.code = code
}

func (self *FuFunction) Code() FuCode {
	return self.code
}

func (self *FuFunction) CheckArgs(args []FuObject) error {
	nargs := len(args)
	if self.minargs == 0 && self.maxargs == 0 && nargs > 0 {
		return fmt.Errorf("function %s takes no arguments (got %d)",
			self, nargs)
	} else if self.minargs == self.maxargs && nargs != self.minargs {
		return fmt.Errorf("function %s takes exactly %d arguments (got %d)",
			self, self.minargs, nargs)
	} else if nargs < self.minargs {
		return fmt.Errorf("function %s requires at least %d arguments (got %d)",
			self, self.minargs, nargs)
	} else if self.maxargs >= 0 && nargs > self.maxargs {
		return fmt.Errorf("function %s takes at most %d arguments (got %d)",
			self, self.maxargs, nargs)
	}
	return nil
}
