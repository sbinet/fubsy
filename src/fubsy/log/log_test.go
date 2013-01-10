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

func Test_TopicNames(t *testing.T) {
	names := TopicNames()
	assert.True(t, len(names) >= 4)
	assert.Equal(t, "ast", names[0])
	assert.Equal(t, "build", names[3])
}

func Test_topic_values(t *testing.T) {
	assert.Equal(t, 1, int(topicnames[0].val))
	assert.Equal(t, 2, int(topicnames[1].val))
	assert.Equal(t, 4, int(topicnames[2].val))
	assert.Equal(t, 8, int(topicnames[3].val))
}

func Test_EnableDebugTopics(t *testing.T) {
	assert.Equal(t, 0, int(defaultlogger.debug))
	EnableDebugTopics([]string{"ast", "build"})
	assert.Equal(t, AST|BUILD, defaultlogger.debug)
	EnableDebugTopics([]string{"  dag", ""})
	assert.Equal(t, AST|DAG|BUILD, defaultlogger.debug)
}

func Test_Logger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	log := New(buf)
	log.Debug(AST, "1: suppressed")
	log.verbosity = 2
	log.Debug(DAG, "2: still suppressed")
	log.verbosity = 3
	log.Debug(DAG, "3: printed")
	assertBuffer(t, "3: printed\n", buf)

	log.verbosity = 0
	log.EnableDebug(AST)
	log.EnableDebug(BUILD)
	log.Debug(AST, "%d: printed", 4)
	log.Debug(DAG, "%d: still suppressed", 5)
	log.Debug(BUILD, "%d: printed", 6)
	assertBuffer(t, "4: printed\n6: printed\n", buf)
}

func Test_Logger_DebugDump(t *testing.T) {
	obj := StubDumper{}
	buf := &bytes.Buffer{}
	log := New(buf)

	log.DebugDump(AST, obj)
	log.DebugDump(BUILD, obj)

	log.EnableDebug(AST)
	log.DebugDump(AST, obj)
	log.DebugDump(BUILD, obj)

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
