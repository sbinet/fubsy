package runtime

import (
	"os"
	"strings"
	"regexp"
	"errors"
	"path/filepath"
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

func NewFileFinder(includes []string) *FuFileFinder {
	return &FuFileFinder{includes: includes}
}

func (self *FuFileFinder) String() string {
	return "<" + strings.Join(self.includes, " ") + ">"
}

func (self *FuFileFinder) Add(other FuObject) (FuObject, error) {
	panic("FileFinder add not implemented yet")
}

func (self *FuFileFinder) typename() string {
	return "file finder"
}

// Walk the filesystem for files matching this FileFinder's include
// patterns. Return the list of matching filenames as a FuList of
// FuString.
func (self *FuFileFinder) Expand(runtime *Runtime) (FuObject, error) {
	result := make(FuList, 0)
	var matches []string
	for _, pattern := range self.includes {
		prefix, tail, err := findRecursive(pattern)
		_ = prefix
		if err != nil {
			return nil, err
		}
		if tail == "" {
			// no recursive patterns here: just do ordinary glob
			matches, err = simpleGlob(pattern)
		} else {
			matches, err = recursiveGlob(prefix, tail)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, makeFuList(matches...)...)
	}
	return result, nil
}

// Scan pattern for valid uses of the recursive glob pattern "**/". If
// exactly one valid pattern is found, return prefix for pattern
// before the "**/" and tail for the part after it. If no recursive
// glob found, return prefix == pattern and tail == "". Otherwise
// return an error describing exactly what is wrong with the pattern.
func findRecursive(pattern string) (prefix, tail string, err error) {
	idx := strings.Index(pattern, "**")
	if idx == -1 {
		prefix = pattern
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
	if idx == 0 {
		prefix = ""
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
	var dirmatches []string
	if prefix == "" {
		dirmatches = []string {"."}
	} else {
		allmatches, err := filepath.Glob(prefix)
		if err != nil {
			return nil, err
		}
		for _, name := range allmatches {
			info, err := os.Stat(name)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				dirmatches = append(dirmatches, name)
			}
		}
	}

	tail, err := translateGlob(tail)
	if err != nil {
		return nil, err
	}
	tailre, err := regexp.Compile(tail)
	if err != nil {
		return nil, err
	}

	// Recursively walk each directory matched by prefix, testing
	// every file found against tail.
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
		if tailre.FindString(relevant) != "" {
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

// translate a Unix wildcard pattern (same syntax as path/filepath.Match())
// to an uncompiled regular expression
func translateGlob(glob string) (string, error) {
	re := []byte {}
	for i := 0; i < len(glob); i++ {
		ch := glob[i]
		switch ch {
		case '*':
			re = append(re, "[^/]*"...)
		case '?':
			re = append(re, "[^/]"...)
		case '.':
			re = append(re, "\\."...)
		case '[':
			//re = append(re, '[')
			for ; i < len(glob) && glob[i] != ']'; i++ {
				re = append(re, glob[i])
			}
			if glob[i] == ']' {
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
