// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package testutils

import (
	"testing"
	"os"
	"path/filepath"
	"io/ioutil"
)

func AssertError(t *testing.T, expect string, actual error) {
	if actual == nil {
		t.Fatal("expected error, but got nil")
	}
	if actual.Error() != expect {
		t.Errorf("expected error message\n%s\nbut got\n%s",
			expect, actual.Error())
	}
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
