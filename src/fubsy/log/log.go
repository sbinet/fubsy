// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package log

// Package log provides a simple logging package for Fubsy. It's
// simpler than Go's standard log package by forcing all output
// options to behave the same (formatted output ending with newline),
// but more complex by adding debug topics and a verbosity level.

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"os"
	"runtime"
	"strings"
)

type Topic uint

const (
	AST Topic = 1 << iota
	DAG
	PLUGINS
	BUILD
	DB
)

type topicname struct {
	val  Topic
	name string
}

var topicnames []topicname

func init() {
	topicnames = []topicname{
		{AST, "ast"},
		{DAG, "dag"},
		{PLUGINS, "plugins"},
		{BUILD, "build"},
		{DB, "db"},
	}
}

func TopicNames() []string {
	names := make([]string, len(topicnames))
	for i, tn := range topicnames {
		names[i] = tn.name
	}
	return names
}

type Logger struct {
	// use a pointer because Logger has a mutex, and mutexes mustn't
	// be copied
	stdlog *stdlog.Logger

	// set of debug topics enabled by the user
	debug Topic

	// verbosity level specified by the user: 0 is quiet, 1 is normal,
	// 2 is verbose, 3 is full debug output
	verbosity uint
}

type Dumper interface {
	Dump(writer io.Writer, indent string)
}

// Print a debugging message related to topic. Used in conjunction
// with the --debug command-line option: e.g. if the user passes
// "--debug=foo,bar" then debug messages with topic "foo" or "bar"
// will be printed; all others wil be suppressed.
func Debug(topic Topic, format string, arg ...interface{}) {
	defaultlogger.Debug(topic, format, arg...)
}

// Print a debugging message related to topic, followed by a stack trace
// that explains how we got to the code that called DebugStack().
func DebugStack(topic Topic, format string, arg ...interface{}) {
	defaultlogger.DebugStack(topic, format, arg...)
}

func DebugDump(topic Topic, object Dumper) {
	defaultlogger.DebugDump(topic, object)
}

func EnableDebugTopics(names []string) error {
	bad := []string{}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		ok := false
		for _, tn := range topicnames {
			if tn.name == name {
				defaultlogger.EnableDebug(tn.val)
				ok = true
				break
			}
		}
		if !ok {
			bad = append(bad, name)
		}
	}
	if len(bad) > 0 {
		return errors.New("invalid debug topic: " + strings.Join(bad, ", "))
	}
	return nil
}

func EnableDebug(topic Topic) {
	defaultlogger.EnableDebug(topic)
}

func SetVerbosity(verbosity uint) {
	defaultlogger.verbosity = verbosity
}

// Print an informative message that is normally suppressed, but shown
// in verbose mode (eg. "-v" on the command line).
func Verbose(format string, arg ...interface{}) {
	defaultlogger.Verbose(format, arg...)
}

// Print an informative message that is normally shown, but suppressed
// in quiet mode (eg. "-q" on the command line).
func Info(format string, arg ...interface{}) {
	defaultlogger.Info(format, arg...)
}

var defaultlogger *Logger

func init() {
	defaultlogger = New(os.Stdout)
}

func New(output io.Writer) *Logger {
	return &Logger{
		stdlog:    stdlog.New(output, "", 0),
		verbosity: 1,
	}
}

func (self *Logger) EnableDebug(topic Topic) {
	self.debug |= topic
}

func (self *Logger) Debug(topic Topic, format string, arg ...interface{}) bool {
	if self.debugEnabled(topic) {
		self.stdlog.Output(2, fmt.Sprintf(format, arg...))
		return true
	}
	return false
}

func (self *Logger) DebugStack(topic Topic, format string, arg ...interface{}) bool {
	if self.Debug(topic, format, arg...) {
		depth := 2
		_, file, line, ok := runtime.Caller(depth)
		for ok {
			fmt.Printf("  from %s:%d\n", file, line)
			depth++
			_, file, line, ok = runtime.Caller(depth)
		}
		return true
	}
	return false
}

func (self *Logger) DebugDump(topic Topic, object Dumper) bool {
	if self.debugEnabled(topic) {
		buf := &bytes.Buffer{}
		object.Dump(buf, "")
		//fmt.Fprintln(buf)
		self.stdlog.Output(2, string(buf.Bytes()))
		return true
	}
	return false
}

func (self *Logger) debugEnabled(topic Topic) bool {
	return self.verbosity >= 3 || (self.debug&topic > 0)
}

func (self *Logger) Verbose(format string, arg ...interface{}) {
	if self.verbosity >= 2 {
		self.stdlog.Output(2, fmt.Sprintf(format, arg...))
	}
}

func (self *Logger) Info(format string, arg ...interface{}) {
	if self.verbosity >= 1 {
		self.stdlog.Output(2, fmt.Sprintf(format, arg...))
	}
}
