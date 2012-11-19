package runtime

import (
	"strings"
)

// file-finding type; another implementation of FuObject, but more
// elaborate than the basic types in basictypes.go
type FuFileFinder struct {
	// include patterns: e.g. for <*.c foo/*.h>, includes will be
	// {"*.c", "foo/*.h"}
	includes []string

	// exclude patterns (can only be added by exclude() method)
	excludes []string
}

func (self *FuFileFinder) String() string {
	return "<" + strings.Join(self.includes, " ") + ">"
}

func (self *FuFileFinder) Add(other FuObject) (FuObject, error) {
	panic("FileFinder add not implemented yet")
}

// Walk the filesystem for files matching this FileFinder's include
// patterns. Return the list of matching filenames.
func (self *FuFileFinder) find() []string {
	result := make([]string, 0)
	return result
}
