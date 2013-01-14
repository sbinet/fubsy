// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
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
	return &BuildRecord{}
}

func (self *BuildRecord) Equal(other *BuildRecord) bool {
	return reflect.DeepEqual(self, other)
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
	if self.ssig == nil {
		self.ssig = make(map[string]([]byte))
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

const FORMAT_VERSION uint32 = 0

// Convert this build record to a binary representation, suitable for
// long-term persistence or wire transmission.
func (self BuildRecord) encode() ([]byte, error) {
	self.check()

	// encoded data format:
	//   version uint32
	//   tsig_len uint32
	//   tsig []byte
	//   num_parents uint32
	//   {
	//     name_len uint32
	//     name []byte        // encoded in UTF-8
	//     ssig_len uint32
	//     ssig []byte
	//   }*                   // repeats num_parents times

	buf := new(bytes.Buffer)
	binary.Write(buf, binary.BigEndian, FORMAT_VERSION)
	binary.Write(buf, binary.BigEndian, uint32(len(self.tsig)))
	binary.Write(buf, binary.BigEndian, self.tsig)
	binary.Write(buf, binary.BigEndian, uint32(len(self.parents)))
	for _, name := range self.parents {
		binary.Write(buf, binary.BigEndian, uint32(len(name)))
		binary.Write(buf, binary.BigEndian, ([]byte)(name))
		ssig := self.ssig[name]
		binary.Write(buf, binary.BigEndian, uint32(len(ssig)))
		binary.Write(buf, binary.BigEndian, ssig)
	}
	return buf.Bytes(), nil
}

func (self *BuildRecord) decode(data []byte) error {
	buf := bytes.NewBuffer(data)
	var num uint32
	binary.Read(buf, binary.BigEndian, &num)
	if num > FORMAT_VERSION {
		return fmt.Errorf("cannot decode build record: encoded version=%d, "+
			"but maximum supported version=%d",
			num, FORMAT_VERSION)
	}

	binary.Read(buf, binary.BigEndian, &num) // length of target signature
	self.tsig = make([]byte, num)
	binary.Read(buf, binary.BigEndian, self.tsig)

	binary.Read(buf, binary.BigEndian, &num) // number of parents
	if num == 0 {
		self.parents = nil
		self.ssig = nil
		return nil
	}

	numparents := int(num)
	self.parents = make([]string, numparents)
	self.ssig = make(map[string][]byte)
	for i := 0; i < numparents; i++ {
		binary.Read(buf, binary.BigEndian, &num) // length of parent name
		bname := make([]byte, num)
		binary.Read(buf, binary.BigEndian, bname)
		self.parents[i] = string(bname)
		binary.Read(buf, binary.BigEndian, &num) // length of source signature
		ssig := make([]byte, num)
		binary.Read(buf, binary.BigEndian, ssig)
		self.ssig[string(bname)] = ssig
	}

	return nil
}

func (self BuildRecord) Dump(writer io.Writer, indent string) {
	fmt.Fprintf(writer, "%starget signature: {%s}\n",
		indent, hex.EncodeToString(self.tsig))
	fmt.Fprintf(writer, "%ssource signatures:\n", indent)
	for _, name := range self.parents {
		sig := hex.EncodeToString(self.ssig[name])
		fmt.Fprintf(writer, "%s  %-40s {%s}\n", indent, name, sig)
	}
}
