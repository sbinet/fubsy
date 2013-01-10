// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"encoding/hex"
	"fmt"
	"io"
)

type SourceRecord struct {
	nodes     []string
	signature map[string]([]byte)
}

func NewSourceRecord() *SourceRecord {
	return &SourceRecord{signature: make(map[string]([]byte))}
}

// Return the list of nodes in this record (by name). Do not modify
// the returned slice; it might share storage with the SourceRecord.
func (self *SourceRecord) Nodes() []string {
	return self.nodes
}

func (self *SourceRecord) AddNode(name string, sig []byte) {
	if sig == nil {
		panic("nil signatures not allowed")
	}
	self.nodes = append(self.nodes, name)
	self.signature[name] = sig
}

// Return the source signature for the specified node in this record,
// or nil if that node is not in this record. (It's impossible to
// store a nil signature.)
func (self SourceRecord) Signature(name string) []byte {
	return self.signature[name]
}

func (self SourceRecord) Dump(writer io.Writer, indent string) {
	for _, name := range self.nodes {
		sig := hex.EncodeToString(self.signature[name])
		fmt.Fprintf(writer, "%s%-40s {%s}\n", indent, name, sig)
	}
}
