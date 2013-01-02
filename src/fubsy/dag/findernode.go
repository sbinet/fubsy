// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"fubsy/types"
)

// Node type that represents filefinders. Code like
//    a = <**/*.c>
// results in one FinderNode being created and assigned to variable
// 'a'. (It's *not* added to the DAG, though, until it's mentioned in
// a build rule.)

type FinderNode struct {
	nodebase

	// include patterns: e.g. for <*.c foo/*.h>, includes will be
	// {"*.c", "foo/*.h"}
	includes []string

	// exclude patterns (can only be added by exclude() method)
	// (currently unused)
	excludes []string
}

func NewFinderNode(includes []string) *FinderNode {
	// XXX what if the include list changes? what about excludes?
	name := strings.Join(includes, "+")
	node := &FinderNode{
		nodebase: makenodebase(name),
		includes: includes,
	}
	return node
}

func MakeFinderNode(dag *DAG, includes []string) *FinderNode {
	node := NewFinderNode(includes)
	node = dag.AddNode(node).(*FinderNode)
	return node
}

func (self *FinderNode) String() string {
	return "<" + strings.Join(self.includes, " ") + ">"
}

func (self *FinderNode) CommandString() string {
	// ummm: what about excludes?
	result := make([]string, len(self.includes))
	for i, pattern := range self.includes {
		result[i] = types.ShellQuote(pattern)
	}
	return strings.Join(result, " ")
}

func (self *FinderNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*FinderNode)
	return (ok &&
		reflect.DeepEqual(self.includes, other.includes) &&
		reflect.DeepEqual(self.excludes, other.excludes))
}

func (self *FinderNode) Add(other_ types.FuObject) (types.FuObject, error) {
	var result types.FuObject
	switch other := other_.(type) {
	case types.FuList:
		// <pat> + ["a", "b", "c"] = [<pat>, "a", "b", "c"]
		list := make(types.FuList, 1+len(other))
		list[0] = self
		copy(list[1:], other)
		result = list
	default:
		// <pat> + anything = [<pat>, anything]
		list := make(types.FuList, 2)
		list[0] = self
		list[1] = other
		result = list
	}
	return result, nil
}

func (self *FinderNode) List() []types.FuObject {
	// You might think it makes sense to return self.includes here,
	// but you'd be wrong. For one thing, that ignores self.excludes.
	// More importantly, a FinderNode is a lazy list of filenames, not
	// a list of patterns. And we should only go expanding the
	// wildcard and searching for filenames when the FinderNode is
	// explicitly Expand()ed, not before. So the only sensible list
	// representation is a singleton.
	return []types.FuObject{self}
}

func (self *FinderNode) Typename() string {
	return "FinderNode"
}

// Walk the filesystem for files matching this FileFinder's include
// patterns. Return the list of matching filenames as a FuList of
// FileNode.
func (self *FinderNode) Expand(ns types.Namespace) (types.FuObject, error) {
	result := make(types.FuList, 0)
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
		for _, filename := range matches {
			result = append(result, newFileNode(filename))
		}
	}
	return result, nil
}

// Node methods

func (self *FinderNode) Exists() (bool, error) {
	// hmmm: it's perfectly meaningful to ask if a FinderNode exists,
	// just unexpected and expensive (have to expand the wildcards)
	panic("Exists() should not be called on a FinderNode " +
		"(graph should have been rebuilt by this point)")
}

func (self *FinderNode) Changed() (bool, error) {
	panic("Changed() should never be called on a FinderNode " +
		"(graph should have been rebuilt by this point)")
}

// Wildcard expansion -- nothing past here has anything to do with
// FuObject, Node, FinderNode, or any of that high-level stuff. It's
// purely about filename patterns and walking the filesystem.

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
	if idx > len(pattern)-4 || pattern[idx+2] != '/' {
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
		prefix = pattern[0 : idx-1]
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
	var choplen int // leading bytes to ignore
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
	re := []byte{}
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
