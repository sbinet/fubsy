package types

import (
)

type Namespace interface {
	Lookup(name string) FuObject
}

// a mapping from name to value, used for a single scope (e.g. phase
// local variables, global variables, ...)
type ValueMap map[string] FuObject

func NewValueMap() ValueMap {
	return make(ValueMap)
}

// Return the object associated with name in this namespace. Return
// nil if no such name defined.
func (self ValueMap) Lookup(name string) FuObject {
	return self[name]
}

// Associate name with value in this namespace.
func (self ValueMap) Assign(name string, value FuObject) {
	self[name] = value
}

// a stack of ValueMaps, which creates a hierarchy of namespaces from
// innermost (eg. local variables) to outermost (global variables)
type ValueStack []ValueMap

func NewValueStack(ns ...ValueMap) ValueStack {
	return ValueStack(ns)
}

func (self *ValueStack) Push(ns ValueMap) {
	*self = append(*self, ns)
}

// Look for the named variable starting in the innermost namespace on
// this stack (most recently pushed). Return the first matching value
// found as we ascend the stack, or nil if name is not defined in any
// namespace on the stack.
func (self ValueStack) Lookup(name string) FuObject {
	var ns ValueMap
	var value FuObject
	for i := len(self) - 1; i >= 0; i-- {
		ns = self[i]
		value = ns.Lookup(name)
		if value != nil {
			return value
		}
	}
	return nil
}

func (self ValueStack) Assign(name string, value FuObject) {
	// walk up the stack and see if name is defined in any existing
	// namespace
	for i := len(self) - 1; i >= 0; i-- {
		ns := self[i]
		if _, ok := ns[name]; ok {
			// found it: replace it here, the innermost namespace
			// where the name is already defined
			ns[name] = value
			return
		}
	}

	// did not find it: add it to the innermost namespace
	self[len(self)-1].Assign(name, value)
}
