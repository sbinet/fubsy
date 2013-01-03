// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package log

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_Logger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New(buf)
	log.Debug("foo", "1: suppressed")
	log.verbosity = 2
	log.Debug("qux", "2: still suppressed")
	log.verbosity = 3
	log.Debug("qux", "3: printed")
	assertBuffer(t, "3: printed\n", buf)

	log.verbosity = 0
	log.debug["foo"] = true
	log.debug["bar"] = true
	log.Debug("foo", "%d: printed", 4)
	log.Debug("qux", "%d: still suppressed", 5)
	log.Debug("bar", "%d: printed", 6)
	log.Debug("barrr", "7: not printed")
	log.Debug("foo.bar", "8: not printed")
	log.Debug("bar.foo", "9: not printed")
	log.Debug("foo.bar.baz", "10: not printed")
	assertBuffer(t, "4: printed\n6: printed\n", buf)
}

func Test_Logger_DebugDump(t *testing.T) {
	obj := StubDumper{}
	buf := &bytes.Buffer{}
	log := New(buf)

	log.DebugDump("foo", obj)
	log.DebugDump("bar", obj)

	log.debug["foo"] = true
	log.DebugDump("foo", obj)
	log.DebugDump("bar", obj)

	assertBuffer(t,
		"this is an object dump\nspread over multiple lines\n",
		buf)
}

type StubDumper struct {
}

func (self StubDumper) Dump(writer io.Writer, indent string) {
	fmt.Fprintln(writer, "this is an object dump")
	fmt.Fprintln(writer, "spread over multiple lines")
}

func Test_Logger_Verbose(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New(buf)

	log.Verbose("yo %s", "dude")
	log.verbosity = 2
	log.Verbose("hey %s", "man")
	log.verbosity = 47
	log.Verbose("this should let %s through", "everything")
	assertBuffer(t, "hey man\nthis should let everything through\n", buf)
}

func Test_Logger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New(buf)

	log.Info("let's try formatting: %v %v %v", 1, true, "meep")
	log.verbosity = 0
	log.Info("suppressed")
	assertBuffer(t, "let's try formatting: 1 true meep", buf)
}

func assertBuffer(t *testing.T, expect string, buf *bytes.Buffer) {
	rbuf := make([]byte, len(expect))
	nbytes, err := buf.Read(rbuf)
	if err != nil {
		t.Error("unexpected error reading from buf: " + err.Error())
		return
	}
	assert.Equal(t, expect, string(rbuf[0:nbytes]))
}
