// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

// Fubsy Node types for filesystem objects

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"syscall"

	"fubsy/types"
)

type FileNode struct {
	// name: filename (relative to top)
	nodebase
}

type GlobNode struct {
	// name: arbitrary unique string
	nodebase
	glob types.FuObject // likely FuFileFinder or FuFinderList
}

// Lookup and return the named file node in dag. If it doesn't exist,
// create a new FileNode, add it to dag, and return it. If it does
// exist but isn't a FileNode, panic.
func MakeFileNode(dag *DAG, name string) *FileNode {
	_, node := dag.addNode(newFileNode(name))
	return node.(*FileNode)
}

func newFileNode(name string) *FileNode {
	return &FileNode{nodebase: makenodebase(name)}
}

func (self *FileNode) Typename() string {
	return "FileNode"
}

func (self *FileNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*FileNode)
	return ok && other.name == self.name
}

func (self *FileNode) Add(other_ types.FuObject) (types.FuObject, error) {
	var result types.FuObject
	switch other := other_.(type) {
	case types.FuString:
		// caller must add it to the appropriate DAG!
		result = newFileNode(self.name + string(other))
	default:
		otherlist := other.List()
		list := make(types.FuList, 1+len(otherlist))
		list[0] = self
		copy(list[1:], otherlist)
		result = list
	}
	return result, nil
}

func (self *FileNode) List() []types.FuObject {
	return []types.FuObject{self}
}

func (self *FileNode) Expand(ns types.Namespace) (types.FuObject, error) {
	return self, nil
}

func (self *FileNode) Exists() (bool, error) {
	info, err := os.Stat(self.name)
	if err != nil {
		errno := err.(*os.PathError).Err.(syscall.Errno)
		if errno == syscall.ENOENT {
			// plain boring old "no such file or directory"
			return false, nil
		} else {
			// some other error
			return false, err
		}
	}

	// This test could be much fancier: do we want an error if a
	// source "file" is really a block device? a FIFO? a symlink?
	if info.IsDir() {
		return false, &os.PathError{
			Op:   "stat",
			Path: self.name,
			Err:  errors.New("is a directory, not a regular file")}
	}
	return true, nil
}

func (self *FileNode) Changed() (bool, error) {
	// placeholder until we have persistent build state
	return true, nil
}

func MakeGlobNode(dag *DAG, glob types.FuObject) *GlobNode {
	_, node := dag.addNode(newGlobNode(globname(glob), glob))
	return node.(*GlobNode)
}

func globname(glob types.FuObject) string {
	// get the address of the underlying struct; panics if glob is not
	// a pointer (roughly speaking), i.e. we are passed an
	// implementation of FuObject that we can't get the address of
	ptr := reflect.ValueOf(glob).Pointer()
	return fmt.Sprintf("glob:%x", ptr)
}

func newGlobNode(name string, glob types.FuObject) *GlobNode {
	return &GlobNode{
		nodebase: makenodebase(name),
		glob:     glob,
	}
}

func (self *GlobNode) Typename() string {
	return "GlobNode"
}

func (self *GlobNode) String() string {
	return self.glob.String()
}

func (self *GlobNode) CommandString() string {
	return self.glob.CommandString()
}

func (self *GlobNode) Equal(other_ types.FuObject) bool {
	other, ok := other_.(*GlobNode)
	return ok && self.glob.Equal(other.glob)
}

func (self *GlobNode) Add(other types.FuObject) (types.FuObject, error) {
	newobj, err := self.glob.Add(other)
	if err != nil {
		return nil, err
	}
	// caller must add it to the DAG (hmmmmm)
	return newGlobNode(globname(other), newobj), nil
}

func (self *GlobNode) List() []types.FuObject {
	return self.glob.List()
}

func (self *GlobNode) Exists() (bool, error) {
	// hmmm: it's perfectly meaningful to ask if a GlobNode exists,
	// just expensive (have to expand the wildcards) and unexpected
	panic("Exists() should not be called on a GlobNode " +
		"(graph should have been rebuilt by this point)")
}

func (self *GlobNode) Expand(ns types.Namespace) (types.FuObject, error) {
	expobj, err := self.glob.Expand(ns)
	if err != nil {
		return nil, err
	}
	filenames := expobj.List()
	newnodes := make(types.FuList, len(filenames))
	for i, fnobj := range filenames {
		// fnobj really should be a FuString
		fn := fnobj.String()
		newnodes[i] = newFileNode(fn)
	}
	return newnodes, nil
}

func (self *GlobNode) Changed() (bool, error) {
	panic("Changed() should never be called on a GlobNode " +
		"(graph should have been rebuilt by this point)")
}
