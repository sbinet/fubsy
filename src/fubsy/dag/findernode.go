// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"fubsy/types"
)

type dirset map[string]bool

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

	// set of directories to prune the walk: if we enter a directory in
	// this set, leave it immediately -- don't look for any files there
	prune dirset

	// cache the result of calling FindFiles(), so subsequent calls
	// are cheap and consistent
	matches []string

	// cache the result of Signature()
	sig []byte
}

func NewFinderNode(includes ...string) *FinderNode {
	// XXX what if the include list changes? what about excludes?
	name := strings.Join(includes, "+")
	node := &FinderNode{
		nodebase: makenodebase(name),
		includes: includes,
	}
	return node
}

func MakeFinderNode(dag *DAG, includes ...string) *FinderNode {
	node := NewFinderNode(includes...)
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
		// <pat> + [a, b, c] = [<pat>, a, b, c]
		// (a, b, c must all be Nodes)
		members := make([]types.FuObject, 1+len(other))
		members[0] = self

		for i, obj := range other {
			switch obj := obj.(type) {
			case types.FuString:
				// <*.c> + ["extra/stuff.c"] should just work
				members[i+1] = NewFileNode(string(obj))
			case Node:
				members[i+1] = obj
			default:
				err := fmt.Errorf(
					"unsupported operation: cannot add list containing "+
						"%s %v to %s %v",
					obj.Typename(), obj, self.Typename(), self)
				return nil, err
			}
		}
		result = newListNode(members...)
	case types.FuString:
		// <pat> + "foo" = [<pat>, FileNode("foo")]
		result = newListNode(self, NewFileNode(string(other)))
	case Node:
		result = newListNode(self, other)
	default:
		err := fmt.Errorf(
			"unsupported operation: cannot add "+
				"%s %v to %s %v",
			other.Typename(), other, self.Typename(), self)
		return nil, err
	}
	return result, nil
}

func (self *FinderNode) List() []types.FuObject {
	// You might think it makes sense to return self.includes here,
	// but you'd be wrong. For one thing, that ignores self.excludes.
	// More importantly, a FinderNode is a lazy list of filenames, not
	// a list of patterns. And we should only go expanding the
	// wildcard and searching for filenames when the FinderNode is
	// explicitly expanded, not before. So the only sensible list
	// representation is a singleton.
	return []types.FuObject{self}
}

func (self *FinderNode) Typename() string {
	return "FinderNode"
}

func (self *FinderNode) copy() Node {
	var c FinderNode = *self
	return &c
}

func (self *FinderNode) NodeExpand(ns types.Namespace) error {
	if self.expanded {
		return nil
	}

	// this does purely textual expansion, e.g. convert
	// <$src/**/*.$ext> to a new FinderNode that will actually find
	// files because '$src' and '$ext' get expanded
	expandlist := func(strings []string) error {
		var err error
		for i, pat := range strings {
			_, strings[i], err = types.ExpandString(pat, ns, nil)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err := expandlist(self.includes)
	if err != nil {
		return err
	}
	err = expandlist(self.excludes)
	if err != nil {
		return err
	}
	self.expanded = true
	return nil
}

// Walk the filesystem for files matching this FinderNode's include
// patterns. Return the list of matching filenames as a FuList of
// FileNode.
func (self *FinderNode) ActionExpand(
	ns types.Namespace, ctx *types.ExpandContext) (
	types.FuObject, error) {

	// if case this node was not already expanded by
	// DAG.ExpandNodes(), do it now so variable references are
	// followed
	var err error
	err = self.NodeExpand(ns)
	if err != nil {
		return nil, err
	}
	filenames, err := self.FindFiles()
	if err != nil {
		return nil, err
	}
	var result types.FuList
	for _, filename := range filenames {
		result = append(result, types.FuString(filename))
	}
	return result, nil
}

// Add dir to the set of prune directories.
func (self *FinderNode) Prune(dir string) {
	if self.prune == nil {
		self.prune = make(dirset)
	}
	self.prune[dir] = true
}

func (self *FinderNode) FindFiles() ([]string, error) {
	if self.matches != nil {
		return self.matches, nil
	}

	var matches []string
	result := []string{}
	for _, pattern := range self.includes {
		recursive, prefix, tail, err := splitPattern(pattern)
		if err != nil {
			return nil, err
		}
		if recursive {
			matches, err = recursiveGlob(self.prune, prefix, tail)
		} else {
			matches, err = simpleGlob(self.prune, pattern)
		}
		if err != nil {
			return nil, err
		}
		result = append(result, matches...)
	}
	self.matches = result
	return result, nil
}

// Node methods

func (self *FinderNode) Exists() (bool, error) {
	filenames, err := self.FindFiles()
	if err != nil {
		return false, err
	}
	return len(filenames) > 0, nil
}

func (self *FinderNode) Signature() ([]byte, error) {
	filenames, err := self.FindFiles()
	if err != nil {
		return nil, err
	}

	// the signature consists of:
	//   sequence of {
	//       filename_hash []byte
	//       file_hash []byte
	//   }
	// this means we can do a simple bytewise comparison to detect
	// change (file added, file removed, file modified), but later we
	// can decode this and figure out *exactly what* changed, for
	// better reporting to the user ("rebuilding x.jar because you
	// added 3 files to <src/x/**/*.java>")

	hash := fnv.New64a()
	sig := make([]byte, 0, (2*hash.Size())*len(filenames))
	for _, filename := range filenames {
		hash.Reset()
		hash.Write(([]byte)(filename))
		sig = hash.Sum(sig)

		hash.Reset()
		err = HashFile(filename, hash)
		if err != nil {
			return nil, err
		}
		sig = hash.Sum(sig)
	}
	return sig, nil

	// // the signature consists of:
	// // - hash(concatenated filenames)
	// // - hash(concatenated file contents)
	// namehash := fnv.New64a()
	// contenthash := fnv.New64a()
	// zero := []byte{0}
	// for _, filename := range filenames {
	// 	namehash.Write(([]byte)(filename))
	// 	namehash.Write(zero)
	// 	err = HashFile(filename, contenthash)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// signature := make([]byte, 0, namehash.Size()+contenthash.Size())
	// signature = namehash.Sum(signature)
	// signature = contenthash.Sum(signature)
	// self.sig = signature
	// return signature, nil
}

// Wildcard expansion -- nothing past here has anything to do with
// FuObject, Node, FinderNode, or any of that high-level stuff. It's
// purely about filename patterns and walking the filesystem.

func (self dirset) String() string {
	keys := make([]string, 0, len(self))
	for key := range self {
		keys = append(keys, key)
	}
	return "{" + strings.Join(keys, ",") + "}"
}

func (self dirset) contains(name string, exact bool) bool {
	if len(self) == 0 {
		return false
	}
	if self[name] {
		return true
	}
	if !exact {
		name = filepath.Dir(name)
		for {
			if self[name] {
				return true
			}
			name = filepath.Dir(name)
			if len(name) == 1 && (name[0] == '.' || name[0] == os.PathSeparator) {
				break
			}
		}
	}
	return false
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

func simpleGlob(prune dirset, pattern string) ([]string, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return matches, err
	}
	result := make([]string, 0, len(matches))
	for _, name := range matches {
		// for non-recursive patterns, pruning just ends up being like
		// an exclude pattern
		if !prune.contains(filepath.Dir(name), false) {
			result = append(result, name)
		}
	}
	return result, nil
}

func recursiveGlob(prune dirset, prefix, tail string) ([]string, error) {
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
	dirmatches := []string{}
	for _, name := range allmatches {
		info, err := os.Stat(name)
		if err != nil {
			return nil, err
		}
		if info.IsDir() && !prune.contains(name, false) {
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
		if prune.contains(path, true) {
			return filepath.SkipDir
		}

		// fail if anything is unreadable (do not silently ignore, unless
		// this directory was pruned)
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
