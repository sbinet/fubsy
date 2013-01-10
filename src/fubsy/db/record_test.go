// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package db

import (
	"bytes"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_Record_basics(t *testing.T) {
	record := NewSourceRecord()
	assert.True(t, record.Signature("foo") == nil)
	assert.True(t, record.Signature("bar") == nil)

	record.AddNode("foo", []byte{0})
	record.AddNode("bar", []byte{})
	sig := record.Signature("foo")
	assert.True(t, sig != nil && len(sig) == 1 && sig[0] == 0)
	sig = record.Signature("bar")
	assert.True(t, sig != nil && len(sig) == 0)
	sig = record.Signature("qux")
	assert.True(t, sig == nil)
}

func Test_Record_Dump(t *testing.T) {
	record := NewSourceRecord()
	record.AddNode("foo/bar/baz", []byte{0x00, 0xff, 0x1e, 0x1f})
	record.AddNode("m! b.*?/...", []byte{})

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
