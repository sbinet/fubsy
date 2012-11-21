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
		message := fmt.Sprintf("unsupported operation: cannot add %s to %s",
			other.typename(), self.typename())
		return nil, RuntimeError{message: message}
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
	panic("list add not implemented yet")
}

func (self FuList) typename() string {
	return "list"
}
