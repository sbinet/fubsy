// The basic Fubsy type system: defines the FuObject interface and
// core implementations of it (FuString, FuList).

package types

import (
	"strings"
	"fmt"
	"reflect"
	"fubsy/dsl"
)

type FuObject interface {
	String() string
	Equal(FuObject) bool
	Add(FuObject) (FuObject, error)

	// Convert an object from its initial form, seen in the main phase
	// (the result of evaluating an expression in the AST), to the
	// final form seen in the build phase. For example, expansion
	// might convert a string "$CC $CFLAGS" to "/usr/bin/gcc -Wall
	// -O2". Expansion can involve conversions within Fubsy's type
	// system: e.g. expanding a FuFileFinder might result in a FuList
	// of file nodes.
	Expand() (FuObject, error)

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

func (self FuString) Equal(other_ FuObject) bool {
	other, ok := other_.(FuString)
	return ok && other == self
}

func (self FuString) Value() string {
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

func (self FuString) Expand() (FuObject, error) {
	// XXX variable expansion!!!
	return self, nil
}

func (self FuString) typename() string {
	return "string"
}


func (self FuList) String() string {
	return "[" + strings.Join(self.Values(), ",") + "]"
}

func (self FuList) Equal(other_ FuObject) bool {
	other, ok := other_.(FuList)
	return ok && reflect.DeepEqual(self, other)
}

func (self FuList) Values() []string {
	result := make([]string, len(self))
	for i, obj := range self {
		result[i] = obj.String()
	}
	return result
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

func (self FuList) Expand() (FuObject, error) {
	result := make(FuList, len(self))
	var err error
	for i, val := range self {
		result[i], err = val.Expand()
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
	return TypeError{message: message}
}

// Convert a variable number of strings to a FuList of FuString.
func makeFuList(strings ...string) FuList {
	result := make(FuList, len(strings))
	for i, s := range strings {
		result[i] = FuString(s)
	}
	return result
}

type TypeError struct {
	location dsl.Location
	message string
}

func (self TypeError) Error() string {
	return self.location.String() + self.message
}
