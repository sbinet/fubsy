// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_Record_basics(t *testing.T) {
	record := NewBuildRecord()
	assert.True(t, record.SourceSignature("foo") == nil)
	assert.True(t, record.SourceSignature("bar") == nil)

	record.AddParent("foo", []byte{0})
	record.AddParent("bar", []byte{})
	sig := record.SourceSignature("foo")
	assert.True(t, sig != nil && len(sig) == 1 && sig[0] == 0)
	sig = record.SourceSignature("bar")
	assert.True(t, sig != nil && len(sig) == 0)
	sig = record.SourceSignature("qux")
	assert.True(t, sig == nil)
}

func Test_Record_encode_decode(t *testing.T) {
	record := NewBuildRecord()
	record.SetTargetSignature([]byte{})
	expect := []byte{
		// all lengths are unsigned big-endian 32-bit integers
		0, 0, 0, 0, // version number
		0, 0, 0, 0, // len() of tsig
		0, 0, 0, 0, // number of parent nodes
	}
	assertEncode(t, expect, record)

	record.SetTargetSignature([]byte{0, 34, 53, 127})
	expect = []byte{
		0, 0, 0, 0, // version number
		0, 0, 0, 4, // len() of tsig
		0, 34, 53, 127, // bytes of tsig
		0, 0, 0, 0, // number of parent nodes
	}
	assertEncode(t, expect, record)

	record.AddParent("foo", []byte{37, 235})
	record.AddParent("bar", []byte{})
	expect = []byte{
		0, 0, 0, 0, // version number
		0, 0, 0, 4, // len() of tsig
		0, 34, 53, 127, // bytes of tsig
		0, 0, 0, 2, // number of parent nodes
		0, 0, 0, 3, // length of first parent name
		'f', 'o', 'o', // name of first parent
		0, 0, 0, 2, // len() of its source sig
		37, 235, // bytes of its source sig
		0, 0, 0, 3, // length of second parent name
		'b', 'a', 'r', // name of second parent
		0, 0, 0, 0, // len() of its source sig
	}
	assertEncode(t, expect, record)
}

func Test_Record_encode_more(t *testing.T) {
	record := NewBuildRecord()
	record.SetTargetSignature([]byte{})
	record.AddParent("node1", []byte{0x5a, 0x8f})
	//record.AddParent("node2", []byte{34})
	expect := []byte{
		0, 0, 0, 0,
		0, 0, 0, 0, // length of tsig
		0, 0, 0, 1, // num parents
		0, 0, 0, 5,
		'n', 'o', 'd', 'e', '1',
		0, 0, 0, 2,
		0x5a, 0x8f,
	}
	assertEncode(t, expect, record)

	record.SetTargetSignature([]byte{0x80, 0x90, 0xA0})
	record.AddParent("node2", []byte{0x34})
	expect = []byte{
		0, 0, 0, 0,
		0, 0, 0, 3, // length of tsig
		0x80, 0x90, 0xA0,
		0, 0, 0, 2, // num parents
		0, 0, 0, 5,
		'n', 'o', 'd', 'e', '1',
		0, 0, 0, 2,
		0x5a, 0x8f,
		0, 0, 0, 5,
		'n', 'o', 'd', 'e', '2',
		0, 0, 0, 1,
		0x34,
	}
	assertEncode(t, expect, record)
}

func assertEncode(t *testing.T, expect []byte, record *BuildRecord) {
	encoded, err := record.encode()
	assert.Nil(t, err)
	if !reflect.DeepEqual(expect, encoded) {
		t.Errorf("expected byte sequence:\n% x\nbut got:\n% x",
			expect, encoded)
	}
	assert.Equal(t, expect, encoded)

	// round-trip it to test decode()
	assertDecode(t, record, encoded)
}

func assertDecode(t *testing.T, expect *BuildRecord, encoded []byte) {
	decoded := NewBuildRecord()
	err := decoded.decode(encoded)
	assert.Nil(t, err)
	if !expect.Equal(decoded) {
		t.Errorf("round-trip encode/decode failed: encoding of\n"+
			"%#v\ndecoded to:\n%#v",
			expect, decoded)
		fmt.Printf("tsig: %v\n", reflect.DeepEqual(expect.tsig, decoded.tsig))
		fmt.Printf("parents: %v\n", reflect.DeepEqual(expect.parents, decoded.parents))
		fmt.Printf("ssig: %v\n", reflect.DeepEqual(expect.ssig, decoded.ssig))
		fmt.Printf("overall: %v\n", reflect.DeepEqual(expect, decoded))
	}
}

func Test_Record_Dump(t *testing.T) {
	record := NewBuildRecord()
	record.AddParent("foo/bar/baz", []byte{0x00, 0xff, 0x1e, 0x1f})
	record.AddParent("m! b.*?/...", []byte{})

	writer := &bytes.Buffer{}
	record.Dump(writer, "%%")
	expect := `
%%foo/bar/baz                              {00ff1e1f}
%%m! b.*?/...                              {}
`[1:]
	actual := string(writer.Bytes())
	if expect != actual {
		t.Errorf("expected:\n%s\nbut got:\n%s", expect, actual)
	}
}
