// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

// In-memory implementation of BuildDB. Fully functional but not
// persistent, so only suitable for use in test code.
type DummyDB struct {
	parents map[string]*SourceRecord
}

func NewDummyDB() *DummyDB {
	return &DummyDB{
		parents: make(map[string]*SourceRecord),
	}
}

func (self *DummyDB) LookupParents(name string) (*SourceRecord, error) {
	match, ok := self.parents[name]
	if !ok {
		return nil, nil
	}
	return match, nil
}

func (self *DummyDB) WriteParents(name string, record *SourceRecord) error {
	self.parents[name] = record
	return nil
}
