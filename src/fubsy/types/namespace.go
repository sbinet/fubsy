// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package types

import (
	"fmt"
	"io"
)

type Namespace interface {
	Lookup(name string) (FuObject, bool)
	Assign(name string, value FuObject)
	Dump(writer io.Writer, indent string)
}

type NamespaceStack interface {
	Namespace
	Push(ns Namespace)
	Pop()
}

// a mapping from name to value, used for a single scope (e.g. phase
// local variables, global variables, ...)
type ValueMap map[string]FuObject

func NewValueMap() ValueMap {
	return make(ValueMap)
}

// Return the object associated with name in this namespace. Return
// ok=false if no such name defined.
func (self ValueMap) Lookup(name string) (value FuObject, ok bool) {
	value, ok = self[name]
	return
}

// Associate name with value in this namespace.
func (self ValueMap) Assign(name string, value FuObject) {
	self[name] = value
}

func (self ValueMap) Dump(writer io.Writer, indent string) {
	for name, val := range self {
		fmt.Fprintf(writer, "%s%s = %T %s\n", indent, name, val, val)
	}
}

// a stack of Namespaces, which creates a hierarchy of namespaces from
// innermost (eg. local variables) to outermost (global variables)
// (N.B. this uses Namespace to keep things nice and abstract, but it
// would be silly to push a ValueStack onto a ValueStack: ValueStack
// in reality is a stack of ValueMaps)
type ValueStack []Namespace

func NewValueStack(ns ...Namespace) ValueStack {
	return ValueStack(ns)
}

func (self *ValueStack) Push(ns Namespace) {
	*self = append(*self, ns)
}

func (self *ValueStack) Pop() {
	*self = (*self)[0 : len(*self)-1]
}

// Look for the named variable starting in the innermost namespace on
// this stack (most recently pushed). Return the first matching value
// found as we ascend the stack, or nil if name is not defined in any
// namespace on the stack.
func (self ValueStack) Lookup(name string) (FuObject, bool) {
	for i := len(self) - 1; i >= 0; i-- {
		ns := self[i]
		if value, ok := ns.Lookup(name); ok {
			return value, true
		}
	}
	return nil, false
}

func (self ValueStack) Assign(name string, value FuObject) {
	// walk up the stack and see if name is defined in any existing
	// namespace
	for i := len(self) - 1; i >= 0; i-- {
		ns := self[i]
		if _, ok := ns.Lookup(name); ok {
			// found it: replace it here, the innermost namespace
			// where the name is already defined
			ns.Assign(name, value)
			return
		}
	}

	// did not find it: add it to the innermost namespace
	self[len(self)-1].Assign(name, value)
}

func (self ValueStack) Dump(writer io.Writer, indent string) {
	for i, ns := range self {
		fmt.Fprintf(writer, "%slevel %d:\n", indent, i)
		ns.Dump(writer, indent+"  ")
	}
}
