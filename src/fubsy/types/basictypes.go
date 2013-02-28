// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// The basic Fubsy type system: defines the FuObject interface and
// core implementations of it (FuString, FuList).

package types

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type FuObject interface {
	// Return a brief, human-readable description of the type of this
	// object. Used in error messages.
	Typename() string

	// Return a string representation of this object for
	// debugging/diagnosis. When feasible, it should return the Fubsy
	// syntax that would reproduce this value, i.e. with
	// quotes/delimiters/escaping that would be accepted by the
	// fubsy/dsl package.
	String() string

	// Return a string representation of this object to use when
	// directly interacting with the OS: e.g. a filename for open() or
	// a command for system().
	ValueString() string

	// Return a string representation of this object that is suitable
	// for use in a shell command. Scalar values should supply quotes
	// so the shell will see them as a single word -- e.g. values with
	// spaces in them must be quoted. Multiple-valued objects should
	// format their values as distinct words, e.g. space-separated
	// with necessary quoting. This is *not* the same as expansion;
	// typically, CommandString() is invoked on expanded values just
	// before incorporating them into a shell command to be executed.
	CommandString() string

	Equal(FuObject) bool
	Add(FuObject) (FuObject, error)

	// Lookup the specified attribute of this object. Return (value,
	// true) if the attribute exists or (*, false) if not. (This is
	// deliberately the same signature as Namespace.Lookup().) Most
	// common use is for looking up methods to call.
	Lookup(name string) (FuObject, bool)

	// Return a slice of FuObjects that you can loop over; intended
	// for easy access to the elements of compound types like FuList.
	// Scalar types (e.g. FuString) should just return themselves in a
	// one-element slice. Callers must not mutate the returned slice,
	// since that might (or might not) affect the original object.
	List() []FuObject

	// Convert a value to the final form needed to run build actions.
	// This happens late in the build phase, after we have determined
	// that a particular target needs to be built, right before
	// actually building it. ActionExpand() expands variable
	// references, so a string "$CC $CFLAGS" might expand to
	// "/usr/bin/gcc -O2 -Wall". ActionExpand() also turns abstract
	// representations of a collection of resources into something
	// that can actually be acted on: canonically, a FinderNode
	// representing <*.c> might expand to a FuList of FuStrings
	// {"foo.c", "bar.c"}. That list in turn will be incorporated into
	// the value being expanded in the appropriate way: if 'src' is
	// the FinderNode, then "cc -c $src" would expand to "cc -c foo.c
	// bar.c". The precise semantics are type-dependent: expanding src
	// in a list ["cc", src] might return ["cc", "foo.c", "bar.c"].
	// ActionExpand() typically returns a value of the receiver's
	// type, e.g. FuString.ActionExpand() returns a FuString, and
	// FuList.ActionExpand() returns a FuList. FinalExpand() never
	// returns nil.
	ActionExpand(ns Namespace, ctx *ExpandContext) (FuObject, error)
}

// a Fubsy string is a Go string, until there's a demonstrated need
// for something more
type FuString struct {
	NullLookupT
	value string
}

func MakeFuString(s string) FuString {
	return FuString{value: s}
}

func (self FuString) Typename() string {
	return "string"
}

func (self FuString) String() string {
	// need to worry about escaping when the DSL supports it!
	return "\"" + self.value + "\""
}

func (self FuString) ValueString() string {
	return self.value
}

func (self FuString) CommandString() string {
	return ShellQuote(self.value)
}

func (self FuString) Equal(other_ FuObject) bool {
	other, ok := other_.(FuString)
	return ok && other == self
}

func (self FuString) Add(other_ FuObject) (FuObject, error) {
	switch other := other_.(type) {
	case FuString:
		// "foo" + "bar" == "foobar"
		return MakeFuString(self.value + other.value), nil
	case FuList:
		// "foo" + ["bar"] == ["foo", "bar"]
		values := make([]FuObject, len(other.values)+1)
		values[0] = self
		copy(values[1:], other.values)
		return MakeFuList(values...), nil
	default:
		return UnsupportedAdd(self, other, "")
	}
	panic("unreachable code")
}

func (self FuString) List() []FuObject {
	return []FuObject{self}
}

var expand_re *regexp.Regexp

func init() {
	// same regex used by the lexer for NAME tokens (no coincidence!)
	namepat := "([a-zA-Z_][a-zA-Z_0-9]*)"
	expand_re = regexp.MustCompile(
		fmt.Sprintf("\\$(?:%s|\\{%s\\})", namepat, namepat))
}

func (self FuString) ActionExpand(ns Namespace, ctx *ExpandContext) (FuObject, error) {
	_, s, err := ExpandString(self.value, ns, ctx)
	if err != nil {
		return nil, err
	}
	return MakeFuString(s), nil
}

// a Fubsy list is a slice of Fubsy objects
type FuList struct {
	NullLookupT
	values []FuObject
}

// Convert a variable number of FuObjects to a FuList.
func MakeFuList(objects ...FuObject) FuList {
	return FuList{values: objects}
}

// Convert a variable number of strings to a FuList of FuString.
func MakeStringList(strings ...string) FuList {
	values := make([]FuObject, len(strings))
	for i, s := range strings {
		values[i] = MakeFuString(s)
	}
	return MakeFuList(values...)
}

func (self FuList) Typename() string {
	return "list"
}

func (self FuList) String() string {
	result := make([]string, len(self.values))
	for i, obj := range self.values {
		result[i] = obj.String()
	}
	return "[" + strings.Join(result, ", ") + "]"
}

func (self FuList) ValueString() string {
	// ValueString() doesn't make a lot of sense for FuList, since it
	// doesn't contain a single filename to open or command to run ...
	// but we have to provide *something*!
	result := make([]string, len(self.values))
	for i, obj := range self.values {
		result[i] = obj.ValueString()
	}
	return strings.Join(result, " ")
}

func (self FuList) CommandString() string {
	result := make([]string, 0, len(self.values))
	for _, val := range self.values {
		csval := val.CommandString()
		if len(csval) > 0 {
			result = append(result, csval)
		}
	}
	return strings.Join(result, " ")
}

func (self FuList) Equal(other_ FuObject) bool {
	other, ok := other_.(FuList)
	return ok && reflect.DeepEqual(self, other)
}

func (self FuList) Add(other FuObject) (FuObject, error) {
	otherlist := other.List()
	values := make([]FuObject, len(self.values)+len(otherlist))
	copy(values, self.values)
	copy(values[len(self.values):], otherlist)
	return MakeFuList(values...), nil
}

func (self FuList) List() []FuObject {
	return self.values
}

func (self FuList) ActionExpand(ns Namespace, ctx *ExpandContext) (FuObject, error) {
	values := make([]FuObject, 0, len(self.values))
	for _, val := range self.values {
		xval, err := val.ActionExpand(ns, ctx)
		if err != nil {
			return nil, err
		}
		values = append(values, xval.List()...)
	}
	return MakeFuList(values...), nil
}

// stub implementation of FuObject (for use in tests)
type StubObject struct {
	name string

	// value returned by ActionExpand() (if nil, return self)
	expansion FuObject

	// so attribute Lookup() works
	ValueMap
}

func NewStubObject(name string, expansion FuObject) StubObject {
	return StubObject{name: name, expansion: expansion}
}

func (self StubObject) Typename() string {
	return "stub"
}

func (self StubObject) String() string {
	return "\"" + self.name + "\""
}

func (self StubObject) ValueString() string {
	return self.name
}

func (self StubObject) CommandString() string {
	return ShellQuote(self.name)
}

func (self StubObject) Equal(other_ FuObject) bool {
	other, ok := other_.(StubObject)
	return ok && other.name == self.name
}

func (self StubObject) Add(other FuObject) (FuObject, error) {
	panic("not implemented")
}

func (self StubObject) List() []FuObject {
	return []FuObject{self}
}

func (self StubObject) ActionExpand(ns Namespace, ctx *ExpandContext) (FuObject, error) {
	if self.expansion == nil {
		return self, nil
	}
	return self.expansion, nil
}

// object passed around when expanding values in order to detect and
// report cyclic variable references nicely
type ExpandContext struct {
	names []string
}

type CyclicReferenceError ExpandContext

func (err CyclicReferenceError) Error() string {
	return "cyclic variable reference: " + strings.Join(err.names, " -> ")
}

// Expand variables in s by looking them up in ns. If s has no
// variable references, just return s; otherwise return a new expanded
// string. Return non-nil error if there are problems expanding the
// string, most likely references to undefined variables.
func ExpandString(s string, ns Namespace, ctx *ExpandContext) (bool, string, error) {
	match := expand_re.FindStringSubmatchIndex(s)
	if match == nil { // fast path for common case
		return false, s, nil
	}

	if ctx == nil {
		ctx = &ExpandContext{}
	}

	pos := 0
	cur := s
	result := ""
	var name string
	var start, end int
	for match != nil {
		group1 := match[2:4] // location of match for "$foo"
		group2 := match[4:6] // location of match for "${foo}"
		if group1[0] > 0 {
			name = cur[group1[0]:group1[1]]
			start = group1[0] - 1
			end = group1[1]
		} else if group2[0] > 0 {
			name = cur[group2[0]:group2[1]]
			start = group2[0] - 2
			end = group2[1] + 1
		} else {
			// this should not happen: panic?
			return false, s, nil
		}

		// check for recursive variable reference
		for _, already := range ctx.names {
			if name == already {
				ctx.names = append(ctx.names, name)
				return false, s, CyclicReferenceError(*ctx)
			}
		}

		value, ok := ns.Lookup(name)
		var cstring string
		if !ok {
			// XXX very similar to error reported by runtime.evaluateName()
			// XXX location?
			err := fmt.Errorf("undefined variable '%s' in string", name)
			return false, s, err
		} else if value != nil {
			ctx.names = append(ctx.names, name)
			xvalue, err := value.ActionExpand(ns, ctx)
			ctx.names = ctx.names[0 : len(ctx.names)-1]
			if err != nil {
				return false, s, err
			} else if xvalue == nil {
				// this violates the contract for FuObject.ActionExpand()
				panic(fmt.Sprintf(
					"value.ActionExpand() returned nil (value == %#v)", value))
			}
			cstring = xvalue.CommandString()
		}

		result += cur[:start] + cstring
		pos = end
		cur = cur[pos:]
		match = expand_re.FindStringSubmatchIndex(cur)
	}
	result += cur
	return true, result, nil
}

const shellmeta = "# `\"'\\&?*[]{}();$><|"

// initialized on demand
var shellreplacer *strings.Replacer

// Return s decorated with quote characters so it can safely be
// included in a shell command.
func ShellQuote(s string) string {
	if len(s) > 0 && !strings.ContainsAny(s, shellmeta) {
		return s // fast path for common case
	}
	double := strings.Contains(s, "\"")
	single := strings.Contains(s, "'")
	if double && single {
		if shellreplacer == nil {
			pairs := make([]string, len(shellmeta)*2)
			for i := 0; i < len(shellmeta); i++ {
				pairs[i*2] = string(shellmeta[i])
				pairs[i*2+1] = "\\" + string(shellmeta[i])
			}
			shellreplacer = strings.NewReplacer(pairs...)
		}
		return shellreplacer.Replace(s)
	} else if single {
		// use double quotes, but be careful of $
		return "\"" + strings.Replace(s, "$", "\\$", -1) + "\""
	} else {
		// use single quotes
		return "'" + s + "'"
	}
	panic("unreachable code")
}
