// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"encoding/hex"
	"fmt"
	"io"
)

// all the information known about a particular target node the last
// time it was successfully built
type BuildRecord struct {
	// signature of the target node itself
	tsig []byte

	// list of parent nodes (sources) from which it was built
	parents []string

	// the signature of each parent node at build time
	ssig map[string]([]byte)
}

func NewBuildRecord() *BuildRecord {
	return &BuildRecord{ssig: make(map[string]([]byte))}
}

func (self *BuildRecord) SetTargetSignature(tsig []byte) {
	self.tsig = tsig
}

// Return the last-known signature of the node whose build is
// described by this record.
func (self BuildRecord) TargetSignature() []byte {
	return self.tsig
}

// Return the list of parents in this record (by name). Do not modify
// the returned slice; it might share storage with the BuildRecord.
func (self *BuildRecord) Parents() []string {
	return self.parents
}

func (self *BuildRecord) AddParent(name string, sig []byte) {
	if sig == nil {
		panic("nil signatures not allowed")
	}
	self.parents = append(self.parents, name)
	self.ssig[name] = sig
}

// Return the source signature for the specified node in this record,
// or nil if that node is not in this record. (It's impossible to
// store a nil signature.)
func (self BuildRecord) SourceSignature(name string) []byte {
	return self.ssig[name]
}

// Panic if this BuildRecord is not in a good state to be written to a
// BuildDB.
func (self BuildRecord) check() {
	if self.tsig == nil {
		panic("BuildRecord: tsig must not be nil")
	}
	if len(self.parents) != len(self.ssig) {
		panic("BuildRecord: parents and ssig must have same length")
	}
	for _, name := range self.parents {
		sig, ok := self.ssig[name]
		if !ok {
			panic("BuildRecord: ssig must have an entry for every parent")
		}
		if sig == nil {
			panic("BuildRecord: every sig in ssig must be non-nil")
		}
	}
}

func (self BuildRecord) Dump(writer io.Writer, indent string) {
	for _, name := range self.parents {
		sig := hex.EncodeToString(self.ssig[name])
		fmt.Fprintf(writer, "%s%-40s {%s}\n", indent, name, sig)
	}
}
