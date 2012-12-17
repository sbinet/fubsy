// Copyright © 2012, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	//"fmt"
	"io/ioutil"
	"os"
	"sort"
	"syscall"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
	"fubsy/types"
)

func Test_defineBuiltins(t *testing.T) {
	ns := types.NewValueMap()
	defineBuiltins(ns)

	fn, ok := ns.Lookup("println")
	assert.True(t, ok)
	assert.NotNil(t, fn)
	assert.Equal(t, fn.(types.FuCallable).Code(), types.FuCode(fn_println))
}

func Test_println(t *testing.T) {
	cleanup1 := testutils.Chtemp()
	defer cleanup1()

	rfile, err := os.Create("stdout")
	if err != nil {
		panic(err)
	}

	// save a copy of stdout in another fd
	stdout_fd := int(os.Stdout.Fd())
	save_stdout, err := syscall.Dup(stdout_fd)
	if err != nil {
		panic(err)
	}

	// redirect stdout to rfile
	err = syscall.Dup2(int(rfile.Fd()), stdout_fd)
	if err != nil {
		panic(err)
	}

	cleanup2 := func() {
		rfile.Close()
		err = syscall.Dup2(save_stdout, stdout_fd)
		if err != nil {
			panic(err)
		}
		syscall.Close(save_stdout)
	}
	defer cleanup2()

	args := types.MakeFuList()
	kwargs := make(map[string]types.FuObject)

	result, err := fn_println(args, kwargs)
	assert.Nil(t, result)
	assert.Nil(t, err)
	data, err := ioutil.ReadFile("stdout")
	assert.Nil(t, err)
	assert.Equal(t, "\n", string(data))
	rfile.Truncate(0)
	rfile.Seek(0, 0)

	args = types.MakeFuList("hello", "world")
	fn_println(args, kwargs)
	data, err = ioutil.ReadFile("stdout")
	assert.Nil(t, err)
	assert.Equal(t, "hello world\n", string(data))
	rfile.Truncate(0)
	rfile.Seek(0, 0)
}

func Test_mkdir(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	// mkdir() happily accepts an empty argument list, to allow for
	// cases where a user-defined list becomes the arg list, and it
	// just happens to be empty
	args := []types.FuObject{}
	kwargs := make(map[string]types.FuObject)
	result, err := fn_mkdir(args, kwargs)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Equal(t, []string{}, dirContents("."))

	// easiest case: create a single dir
	args = types.MakeFuList("foo")
	result, err = fn_mkdir(args, kwargs)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Equal(t, []string{"foo"}, dirContents("."))
	assert.True(t, isDir("foo"))

	// create multiple dirs, including "foo" which already exists
	args = types.MakeFuList("meep/meep/meep", "foo", "meep/beep")
	result, err = fn_mkdir(args, kwargs)
	assert.Nil(t, result)
	assert.Nil(t, err)
	assert.Equal(t, []string{"foo", "meep"}, dirContents("."))
	assert.True(t, isDir("foo"))
	assert.True(t, isDir("meep/meep"))
	assert.True(t, isDir("meep/meep/meep"))
	assert.True(t, isDir("meep/beep"))

	// now with an error in the middle of the list (*but* we still
	// create the other requested dirs!)
	testutils.TouchFiles("meep/zap")
	args = types.MakeFuList("meep/bap", "meep/zap/zip", "foo/bar")
	result, err = fn_mkdir(args, kwargs)
	assert.Nil(t, result)
	assert.Equal(t, "mkdir meep/zap: not a directory", err.Error())
	assert.True(t, isDir("meep/bap"))
	assert.True(t, isDir("foo/bar"))

	// finally, with multiple errors
	args = append(args, types.FuString("meep/zap/blop"))
	result, err = fn_mkdir(args, kwargs)
	assert.Nil(t, result)
	assert.Equal(t,
		"error creating 2 directories:\n"+
			"  mkdir meep/zap: not a directory\n"+
			"  mkdir meep/zap: not a directory",
		err.Error())
}

func dirContents(dir string) []string {
	f, err := os.Open(dir)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		panic(err)
	}
	sort.Strings(names)
	return names
}

func isDir(name string) bool {
	st, err := os.Stat(name)
	if err != nil {
		panic(err)
	}
	return st.IsDir()
}
