package runtime

import (
	"strings"
)

// a Fubsy filefinder (<**/*.c>) is a little more elaborate
type FuFileFinder struct {
	// include patterns: e.g. for <*.c foo/*.h>, includes will be
	// {"*.c", "foo/*.h"}
	includes []string

	// exclude patterns (can only be added by exclude() method)
	excludes []string

	// the list of files found when this FuFileFinder was actually
	// executed (as late as possible)
	result []string
}

func (self *FuFileFinder) String() string {
	return "<" + strings.Join(self.includes, " ") + ">"
}

func (self *FuFileFinder) Add(other FuObject) FuObject {
	panic("FileFinder add not implemented yet")
}
