// The basic Fubsy type system: defines the FuObject interface and
// core implementations of it (FuString, FuList).

package runtime

import (
	"strings"
)

type FuObject interface {
	String() string
	Add(FuObject) (FuObject, error)
}

// a Fubsy string is a Go string, until there's a demonstrated need
// for something more
type FuString string

// a Fubsy list is a slice of Fubsy objects
type FuList []FuObject


func (self FuString) String() string {
	return string(self)
}

func (self FuString) Add(other FuObject) (FuObject, error) {
	panic("string add not implemented yet")
}


func (self FuList) String() string {
	result := make([]string, len(self))
	for i, obj := range self {
		result[i] = obj.String()
	}
	return "[" + strings.Join(result, ",") + "]"
}

func (self FuList) Add() (FuObject, error) {
	panic("list add not implemented yet")
}
