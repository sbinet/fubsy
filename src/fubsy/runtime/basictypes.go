// The basic Fubsy type system: defines the FuObject interface and
// core implementations of it (FuString, FuList).

package runtime

import (
	"strings"
	"fmt"
)

type FuObject interface {
	String() string
	Add(FuObject) (FuObject, error)

	// Convert an object from its initial form, seen in the main phase
	// (the result of evaluating an expression in the AST), to the
	// final form seen in the build phase. For example, expansion
	// might convert a string "$CC $CFLAGS" to "/usr/bin/gcc -Wall
	// -O2". Expansion can involve conversions within Fubsy's type
	// system: e.g. expanding a FuFileFinder might result in a FuList
	// of file nodes.
	Expand(runtime *Runtime) (FuObject, error)

	// Return a brief, human-readable description of the type of this
	// object. Used in error messages.
	typename() string
}

// a Fubsy string is a Go string, until there's a demonstrated need
// for something more
type FuString string

// a Fubsy list is a slice of Fubsy objects
type FuList []FuObject


func (self FuString) String() string {
	return string(self)
}

func (self FuString) Add(other_ FuObject) (FuObject, error) {
	switch other := other_.(type) {
	case FuString:
		// "foo" + "bar" == "foobar"
		return FuString(self + other), nil
	case FuList:
		// "foo" + ["bar"] == ["foo", "bar"]
		newlist := make(FuList, len(other) + 1)
		newlist[0] = self
		copy(newlist[1:], other)
		return newlist, nil
	default:
		return nil, unsupportedOperation(self, other, "cannot add %s to %s")
	}
	panic("unreachable code")
}

func (self FuString) Expand(runtime *Runtime) (FuObject, error) {
	// XXX variable expansion!!!
	return self, nil
}

func (self FuString) typename() string {
	return "string"
}


func (self FuList) String() string {
	result := make([]string, len(self))
	for i, obj := range self {
		result[i] = obj.String()
	}
	return "[" + strings.Join(result, ",") + "]"
}

func (self FuList) Add(other_ FuObject) (FuObject, error) {
	switch other := other_.(type) {
	case FuList:
		// ["foo", "bar"] + ["qux"] == ["foo", "bar", "qux"]
		newlist := make(FuList, len(self) + len(other))
		copy(newlist, self)
		copy(newlist[len(self):], other)
		return newlist, nil
	case FuString:
		// ["foo", "bar"] + "qux" == ["foo", "bar", "qux"]
		newlist := make(FuList, len(self) + 1)
		copy(newlist, self)
		newlist[len(self)] = other
		return newlist, nil
	default:
		return nil, unsupportedOperation(self, other, "cannot add %s to %s")
	}
	panic("unreachable code")
}

func (self FuList) Expand(runtime *Runtime) (FuObject, error) {
	result := make(FuList, len(self))
	var err error
	for i, val := range self {
		result[i], err = val.Expand(runtime)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (self FuList) typename() string {
	return "list"
}

func unsupportedOperation(self FuObject, other FuObject, detail string) error {
	message := fmt.Sprintf("unsupported operation: " + detail,
		other.typename(), self.typename())
	return RuntimeError{message: message}
}

// Convert a variable number of strings to a FuList of FuString.
func makeFuList(strings ...string) FuList {
	result := make(FuList, len(strings))
	for i, s := range strings {
		result[i] = FuString(s)
	}
	return result
}
