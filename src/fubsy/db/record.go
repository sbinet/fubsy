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

func (self *SourceRecord) AddNode(name string, sig []byte) {
	self.nodes = append(self.nodes, name)
	self.signature[name] = sig
}

func (self SourceRecord) Contains(name string) bool {
	_, ok := self.signature[name]
	return ok
}

func (self SourceRecord) Dump(writer io.Writer, indent string) {
	for _, name := range self.nodes {
		sig := hex.EncodeToString(self.signature[name])
		fmt.Fprintf(writer, "%s%-40s {%s}\n", indent, name, sig)
	}
}
