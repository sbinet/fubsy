// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"fmt"
	"io"
)

// In-memory implementation of BuildDB. Fully functional but not
// persistent, so only suitable for use in test code.
type FakeDB struct {
	parents map[string]*BuildRecord
}

func NewFakeDB() *FakeDB {
	return &FakeDB{
		parents: make(map[string]*BuildRecord),
	}
}

func (self *FakeDB) Close() error {
	return nil
}

func (self *FakeDB) LookupNode(name string) (*BuildRecord, error) {
	match, ok := self.parents[name]
	if !ok {
		return nil, nil
	}
	return match, nil
}

func (self *FakeDB) WriteNode(name string, record *BuildRecord) error {
	record.check()
	self.parents[name] = record
	return nil
}

func (self *FakeDB) Dump(writer io.Writer, indent string) {
	for node, record := range self.parents {
		fmt.Fprintf(writer, "%s%s:\n", indent, node)
		record.Dump(writer, indent+"  ")
	}
}
