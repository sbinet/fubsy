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

func (self *FileNode) Equal(other_ Node) bool {
	other, ok := other_.(*FileNode)
	return ok && other.name == self.name
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
	// get the address of the underlying struct; panics if glob is not
	// a pointer (roughly speaking), i.e. we are passed an
	// implementation of FuObject that we can't get the address of
	ptr := reflect.ValueOf(glob).Pointer()
	name := fmt.Sprintf("glob:%x", ptr)
	_, node := dag.addNode(newGlobNode(name, glob))
	return node.(*GlobNode)
}

func newGlobNode(name string, glob types.FuObject) *GlobNode {
	return &GlobNode{
		nodebase: makenodebase(name),
		glob:     glob,
	}
}

func (self *GlobNode) String() string {
	return self.glob.String()
}

func (self *GlobNode) Equal(other_ Node) bool {
	other, ok := other_.(*GlobNode)
	return ok && self.glob.Equal(other.glob)
}

func (self *GlobNode) Exists() (bool, error) {
	// hmmm: it's perfectly meaningful to ask if a GlobNode exists,
	// just expensive (have to expand the wildcards) and unexpected
	panic("Exists() should not be called on a GlobNode " +
		"(graph should have been rebuilt by this point)")
}

func (self *GlobNode) Expand(dag *DAG, ns types.Namespace) ([]Node, error) {
	filenames, err := self.glob.Expand(ns)
	if err != nil {
		return nil, err
	}
	newnodes := []Node{}
	for _, fnobj := range filenames.List() {
		// fnobj must be a FuString -- panic if not
		fn := fnobj.(types.FuString).Value()
		newnodes = append(newnodes, newFileNode(fn))
	}
	return newnodes, nil
}

func (self *GlobNode) Changed() (bool, error) {
	panic("Changed() should never be called on a GlobNode " +
		"(graph should have been rebuilt by this point)")
}
