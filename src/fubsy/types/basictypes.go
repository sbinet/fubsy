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
	String() string

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

	// Return a slice of FuObjects that you can loop over; intended
	// for easy access to the elements of compound types like FuList.
	// Scalar types (e.g. FuString) should just return themselves in a
	// one-element slice. Callers must not mutate the returned slice,
	// since that might (or might not) affect the original object.
	List() []FuObject

	// Convert an object from its initial form, seen in the main phase
	// (the result of evaluating an expression in the AST), to the
	// final form seen in the build phase. For example, expansion
	// might convert a string "$CC $CFLAGS" to "/usr/bin/gcc -Wall
	// -O2". Expansion can involve conversions within Fubsy's type
	// system: e.g. expanding a FinderNode might result in a FuList of
	// file nodes.
	Expand(ns Namespace) (FuObject, error)

	// Return a brief, human-readable description of the type of this
	// object. Used in error messages.
	Typename() string
}

// a Fubsy string is a Go string, until there's a demonstrated need
// for something more
type FuString string

// a Fubsy list is a slice of Fubsy objects
type FuList []FuObject

func (self FuString) String() string {
	return string(self)
}

func (self FuString) CommandString() string {
	return ShellQuote(string(self))
}

func (self FuString) Equal(other_ FuObject) bool {
	other, ok := other_.(FuString)
	return ok && other == self
}

func (self FuString) Add(other_ FuObject) (FuObject, error) {
	switch other := other_.(type) {
	case FuString:
		// "foo" + "bar" == "foobar"
		return FuString(self + other), nil
	case FuList:
		// "foo" + ["bar"] == ["foo", "bar"]
		newlist := make(FuList, len(other)+1)
		newlist[0] = self
		copy(newlist[1:], other)
		return newlist, nil
	default:
		return nil, unsupportedOperation(self, other, "cannot add %s to %s")
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

func (self FuString) Expand(ns Namespace) (FuObject, error) {

	match := expand_re.FindStringSubmatchIndex(string(self))
	if match == nil { // fast path for common case
		return self, nil
	}

	pos := 0
	cur := string(self)
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
			return self, nil
		}

		value, ok := ns.Lookup(name)
		var cstring string
		if !ok {
			// XXX very similar to error reported by runtime.evaluateName()
			// XXX location?
			return self, fmt.Errorf("undefined variable '%s' in string", name)
		} else if value != nil {
			xvalue, err := value.Expand(ns)
			if err != nil {
				return nil, err
			}
			if xvalue == nil {
				// this violates the contract for FuObject.Expand()
				panic(fmt.Sprintf(
					"value.Expand() returned nil (value == %#v)", value))
			}
			cstring = xvalue.CommandString()
		}

		result += cur[:start] + cstring
		pos = end
		cur = cur[pos:]
		match = expand_re.FindStringSubmatchIndex(cur)
	}
	result += cur
	return FuString(result), nil
}

func (self FuString) Typename() string {
	return "string"
}

func (self FuList) String() string {
	result := make([]string, len(self))
	for i, obj := range self {
		result[i] = obj.String()
	}
	return "[" + strings.Join(result, ",") + "]"
}

func (self FuList) CommandString() string {
	result := make([]string, 0, len(self))
	for _, val := range self {
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
	result := make(FuList, len(self)+len(otherlist))
	copy(result, self)
	copy(result[len(self):], otherlist)
	return result, nil
}

func (self FuList) List() []FuObject {
	return self
}

func (self FuList) Expand(ns Namespace) (FuObject, error) {
	result := make(FuList, 0, len(self))
	for _, val := range self {
		xval, err := val.Expand(ns)
		if err != nil {
			return nil, err
		}
		result = append(result, xval.List()...)
	}
	return result, nil
}

func (self FuList) Typename() string {
	return "list"
}

func unsupportedOperation(self FuObject, other FuObject, detail string) error {
	return fmt.Errorf("unsupported operation: "+detail,
		other.Typename(), self.Typename())
}

// Convert a variable number of strings to a FuList of FuString.
func MakeFuList(strings ...string) FuList {
	result := make(FuList, len(strings))
	for i, s := range strings {
		result[i] = FuString(s)
	}
	return result
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
