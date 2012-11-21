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

func (self FuList) typename() string {
	return "list"
}

func unsupportedOperation(self FuObject, other FuObject, detail string) error {
	message := fmt.Sprintf("unsupported operation: " + detail,
		other.typename(), self.typename())
	return RuntimeError{message: message}
}
