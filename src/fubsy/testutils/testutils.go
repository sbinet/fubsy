// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package testutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Ensure that errs is empty. If not, fail the current test
// and return false. Otherwise return true.
func NoErrors(t *testing.T, errs []error) {
	if len(errs) == 0 {
		return
	}
	formatted := make([]string, len(errs))
	for i, err := range errs {
		formatted[i] = fmt.Sprintf("%T: %s", err, err.Error())
	}
	t.Fatalf("%sexpected no errors, but got %d:\n%s",
		caller(1), len(errs), strings.Join(formatted, "\n"))
}

// Ensure that err is nil. If not, fail the current test and return
// false; otherwise return true.
func NoError(t *testing.T, err error) {
	if err == nil {
		return
	}
	t.Fatalf("%sexpected no error, but got %T: %s", caller(1), err, err.Error())
}

func caller(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if ok {
		return fmt.Sprintf("%s:%d: ", filepath.Base(file), line)
	}
	return ""
}

// Create a temporary directory. Return the name of the directory and
// a function to clean it up when you're done with it. Panics on any
// error (as does the cleanup function).
func Mktemp() (tmpdir string, cleanup func()) {
	tmpdir, err := ioutil.TempDir("", "fubsytest.")
	if err != nil {
		panic(err)
	}
	cleanup = func() {
		err := os.RemoveAll(tmpdir)
		if err != nil {
			panic(err)
		}
	}
	return
}

// Create a temporary directory and chdir to it. Returns a function
// that chdirs back to your original location and removes the temp
// directory. Panics on any error (as does the goback function).
func Chtemp() (goback func()) {
	tmpdir, cleanup := Mktemp()
	orig, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	err = os.Chdir(tmpdir)
	if err != nil {
		panic(err)
	}

	goback = func() {
		err := os.Chdir(orig)
		if err != nil {
			panic(err)
		}
		cleanup()
	}
	return
}

// Create a file in tmpdir and write data to it. Panics on any error.
func Mkfile(tmpdir string, basename string, data string) string {
	fn := filepath.Join(tmpdir, basename)
	err := ioutil.WriteFile(fn, []byte(data), 0644)
	if err != nil {
		panic(err)
	}
	return fn
}

// Create many empty files. Creates directories to contains those
// files as needed. Panics on any error.
func TouchFiles(filenames ...string) {
	for _, fn := range filenames {
		err := os.MkdirAll(filepath.Dir(fn), 0755)
		if err != nil {
			panic(err)
		}
		file, err := os.Create(fn)
		if err != nil {
			panic(err)
		}
		file.Close()
	}
}

// Create many directories. Panics on any error.
// XXX currently unused: if still unused by 2013-06-30, delete it.
func Mkdirs(dirs ...string) {
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(err)
		}
	}
}

// Modify the permissions of the specified file so that it's
// inaccessible (no read, no write, no execute) to all users. Panic on
// error.
func ChmodNoAccess(name string) {
	chmodMask(name, os.ModePerm, 0)
}

func ChmodRO(name string) {
	chmodMask(name, os.FileMode(0222), 0)
}

// Modify the permissions of the specified file so that it's
// readable/writeable/executable by the owner.
func ChmodOwnerAll(name string) {
	chmodMask(name, 0, 0700)
}

func chmodMask(name string, offbits, onbits os.FileMode) {
	// hmmm: does this work on windows?
	info, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	mode := (info.Mode() & ^offbits) | onbits
	err = os.Chmod(name, mode)
	if err != nil {
		panic(err)
	}
}
