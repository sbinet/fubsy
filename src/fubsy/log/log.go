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
	"fmt"
	"io"
	stdlog "log"
	"os"
)

type Logger struct {
	// use a pointer because Logger has a mutex, and mutexes mustn't
	// be copied
	stdlog *stdlog.Logger

	// set of debug topics enabled by the user
	debug map[string]bool

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
func Debug(topic string, format string, arg ...interface{}) {
	defaultlogger.Debug(topic, format, arg...)
}

func DebugDump(topic string, object Dumper) {
	defaultlogger.DebugDump(topic, object)
}

func EnableDebug(topic string) {
	defaultlogger.debug[topic] = true
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
		debug:     make(map[string]bool),
		verbosity: 1,
	}
}

func (self *Logger) Debug(topic string, format string, arg ...interface{}) {
	if self.verbosity >= 3 || self.debug[topic] {
		self.stdlog.Output(2, fmt.Sprintf(format, arg...))
	}
}

func (self *Logger) DebugDump(topic string, object Dumper) {
	if self.verbosity >= 3 || self.debug[topic] {
		var buf bytes.Buffer
		object.Dump(&buf, "")
		self.stdlog.Output(2, string(buf.Bytes()))
	}
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
