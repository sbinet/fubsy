package types

import (
	"os"
	"strings"
	"regexp"
	"errors"
	"reflect"
	"path/filepath"
)

// file-finding type; another implementation of FuObject, but more
// elaborate than the basic types in basictypes.go
type FuFileFinder struct {
	id int

	// include patterns: e.g. for <*.c foo/*.h>, includes will be
	// {"*.c", "foo/*.h"}
	includes []string

	// exclude patterns (can only be added by exclude() method)
	excludes []string
}

// Object that results from adding file finders together: e.g.
//   <*.c> + <*.h> + <**/*.java>
// evaluates to a FuFinderList with three elements. Expanding
// the FuFinderList expands each of its elements in turn.
type FuFinderList struct {
	id int
	elements []*FuFileFinder
}

var ffid int = 0

func NewFileFinder(includes []string) *FuFileFinder {
	ff := &FuFileFinder{id: ffid, includes: includes}
	ffid++
	return ff
}

func (self *FuFileFinder) Id() int {
	return self.id
}

func (self *FuFileFinder) String() string {
	return "<" + strings.Join(self.includes, " ") + ">"
}

func (self *FuFileFinder) Equal(other_ FuObject) bool {
	other, ok := other_.(*FuFileFinder)
	return (ok &&
		reflect.DeepEqual(self.includes, other.includes) &&
		reflect.DeepEqual(self.excludes, other.excludes))
}

func (self *FuFileFinder) Add(other_ FuObject) (FuObject, error) {
	result := NewFinderList()
	result.elements = []*FuFileFinder {self}
	switch other := other_.(type) {
	case *FuFileFinder:
		// <p1> + <p2>
		result.elements = append(result.elements, other)
	case *FuFinderList:
		// <p1> + (<p2> + <p3>) (evaluating the outer +)
		result.elements = append(result.elements, other.elements...)
	default:
		return nil, unsupportedOperation(self, other, "cannot add %s to %s")
	}
	return result, nil

	// result := NewFileFinder(self.includes)
	// result.excludes = self.excludes
	// result.chain = self.chain
	// prev := result
	// for cur := self.chain; cur != nil; cur = cur.chain {
	// 	prev = cur
	// }
	// prev.chain = other
	// return result, nil
}

func (self *FuFileFinder) typename() string {
	return "file finder"
}

// Walk the filesystem for files matching this FileFinder's include
// patterns. Return the list of matching filenames as a FuList of
// FuString.
// XXX this has to happen in the main phase, so using Expand() for this purpose is wrong!!!
func (self *FuFileFinder) Expand() (FuObject, error) {
	result := make(FuList, 0)
	var matches []string
	for _, pattern := range self.includes {
		recursive, prefix, tail, err := splitPattern(pattern)
		if err != nil {
			return nil, err
		}
		if recursive {
			matches, err = recursiveGlob(prefix, tail)
		} else {
			matches, err = simpleGlob(pattern)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, makeFuList(matches...)...)
	}
	return result, nil
}

// Scan pattern for the recursive glob pattern "**". If any are found,
// return recursive = true, prefix = pattern up to the first "**/" and
// tail = the part after it. If no recursive glob found, return
// recursive = false. Otherwise return an error describing exactly
// what is wrong with the pattern.
func splitPattern(pattern string) (
	recursive bool,
	prefix, tail string,
	err error) {
	idx := strings.Index(pattern, "**")
	if idx == -1 {
		recursive = false
		return
	}
	if idx > 0 && pattern[idx-1] != '/' {
		// XXX assumes patterns have been normalized to Unix syntax
		err = errors.New(
			"recursive glob pattern ** may only occur " +
			"at the start of a pattern or immediately after /")
		return
	}
	if idx > len(pattern) - 4 || pattern[idx+2] != '/' {
		// the minimum valid pattern is "**/x": "**/" and "**" are invalid
		err = errors.New(
			"recursive glob pattern ** must be followed " +
			"by / and at least one more character")
		return
	}
	recursive = true
	if idx == 0 {
		prefix = "."
	} else {
		prefix = pattern[0:idx-1]
	}
	tail = pattern[idx+3:]
	return
}

func simpleGlob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func recursiveGlob(prefix, tail string) ([]string, error) {
	// prefix might be "", "foo", "fo?", or "fo?/*/b*r": let
	// filepath.Glob() find all matching filenames, and then reduce
	// the list to matching directories

	// XXX using filepath.Glob() means that sometimes we allow \ as an
	// escape character, and sometimes we don't: I suspect we're just
	// gonna have to reimplement filepath.Glob() and friends to get
	// exactly the syntax we want ;-(

	allmatches, err := filepath.Glob(prefix)
	if err != nil {
		return nil, err
	}
	var dirmatches []string
	for _, name := range allmatches {
		info, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			dirmatches = append(dirmatches, name)
		}
	}

	tail, err = translateGlob(tail)
	if err != nil {
		return nil, err
	}
	tailre, err := regexp.Compile(tail)
	if err != nil {
		return nil, err
	}

	// Recursively walk each directory matched by prefix, testing
	// every file found against tailre.
	var curdir string
	var matches []string
	var choplen int				// leading bytes to ignore
	visit := func(path string, info os.FileInfo, err error) error {
		// fail if anything is unreadable (do not silently ignore)
		if err != nil {
			return err
		}
		if path == curdir {
			// ignore starting point of this walk
			return nil
		}

		relevant := path[choplen:]
		if !info.IsDir() && tailre.FindString(relevant) != "" {
			matches = append(matches, path)
		}
		return nil
	}

	for _, curdir = range dirmatches {
		if curdir == "." {
			// filepath.Walk() conveniently drops the leading "./"
			choplen = 0
		} else {
			choplen = len(curdir) + 1
		}
		err := filepath.Walk(curdir, visit)
		if err != nil {
			return nil, err
		}
	}

	return matches, nil
}

// Translate a Unix wildcard pattern to a regular expression (caller
// must compile it). Syntax:
// - "*" matches zero or more non-separator characters
//   (where separator is platform-dependent / or \)
// - "?" matches exactly one non-separator character
// - "[<range>]" matches exactly one character in <range> (using
//   RE2 regex syntax)
// - "**" matches zero or more characters (including separators) --
//   effectively a recursive search
func translateGlob(glob string) (string, error) {
	re := []byte {}
	for i := 0; i < len(glob); i++ {
		ch := glob[i]
		switch ch {
		case '*':
			if i+1 < len(glob) && glob[i+1] == '*' {
				re = append(re, ".*"...)
				i++
			} else {
				re = append(re, "[^/]*"...)
			}
		case '?':
			re = append(re, "[^/]"...)
		case '.':
			re = append(re, "\\."...)
		case '[':
			for ; i < len(glob) && glob[i] != ']'; i++ {
				re = append(re, glob[i])
			}
			if i < len(glob) {
				re = append(re, ']')
			} else {
				return "", errors.New("unterminated character range")
			}
		default:
			re = append(re, ch)
		}
	}
	re = append(re, '$')
	return string(re), nil
}

func NewFinderList() *FuFinderList {
	fl := &FuFinderList{id: ffid}
	ffid++
	return fl
}

func (self *FuFinderList) Id() int {
	return self.id
}

func (self *FuFinderList) typename() string {
	return "file finder"		// hmmm: ambiguous, but clearer errors?
}

func (self *FuFinderList) String() string {
	result := make([]string, len(self.elements))
	for i, finder := range self.elements {
		result[i] = finder.String()
	}
	return strings.Join(result, " + ")
}

func (self *FuFinderList) Equal(other_ FuObject) bool {
	other, ok := other_.(*FuFinderList)
	return ok && reflect.DeepEqual(self.elements, other.elements)
}

func (self *FuFinderList) Add(other_ FuObject) (FuObject, error) {
	result := &FuFinderList{}
	result.elements = make([]*FuFileFinder, len(self.elements))
	copy(result.elements, self.elements)

	switch other := other_.(type) {
	case *FuFileFinder:
		result.elements = append(result.elements, other)
	case *FuFinderList:
		result.elements = append(result.elements, other.elements...)
	default:
		return nil, unsupportedOperation(self, other, "cannot add %s to %s")
	}
	return result, nil
}

func (self *FuFinderList) Expand() (FuObject, error) {
	result := make(FuList, 0)
	for _, element := range self.elements {
		batch, err := element.Expand()
		if err != nil {
			return nil, err
		}
		result = append(result, batch.(FuList)...)
	}
	return result, nil
}
