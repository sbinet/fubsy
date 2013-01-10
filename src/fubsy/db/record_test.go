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
