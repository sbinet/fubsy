// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package dag

// Fubsy Node types for filesystem objects

import (
	"errors"
	"hash/fnv"
	"io"
	"os"
	"syscall"

	"fubsy/types"
)

type FileNode struct {
	// name: filename (relative to top)
	nodebase

	// cache the signature so we only compute it once per process
	sig []byte
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

func (self *FileNode) copy() Node {
	var c FileNode = *self
	return &c
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

// temporary until Signature() returns something useful
func (self *FileNode) Changed(oldsig, newsig []byte) bool {
	return true
}

func (self *FileNode) Signature() ([]byte, error) {
	if self.sig != nil {
		return self.sig, nil
	}
	file, err := os.Open(self.Name())
	if err != nil {
		return nil, err
	}
	hash := fnv.New64a()
	block := hash.BlockSize()
	data := make([]byte, block)
	for {
		nbytes, err := file.Read(data)
		if nbytes == 0 && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		hash.Write(data[0:nbytes])
	}
	signature := make([]byte, 0, hash.Size())
	signature = hash.Sum(signature)
	self.sig = signature
	return signature, nil
}
