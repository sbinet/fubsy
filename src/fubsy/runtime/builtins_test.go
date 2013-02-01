// Copyright © 2012-2013, Greg Ward. All rights reserved.
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

	"fubsy/dag"
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

	fn, ok = ns.Lookup("remove")
	assert.True(t, ok)
	assert.NotNil(t, fn)
	assert.Equal(t, fn.(types.FuCallable).Code(), types.FuCode(fn_remove))
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

	args := FunctionArgs{
		args:   types.MakeFuList(),
		kwargs: make(map[string]types.FuObject),
	}

	result, errs := fn_println(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	data, err := ioutil.ReadFile("stdout")
	assert.Nil(t, err)
	assert.Equal(t, "\n", string(data))
	rfile.Truncate(0)
	rfile.Seek(0, 0)

	args.args = types.MakeFuList("hello", "world")
	fn_println(args)
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
	args := FunctionArgs{
		args:   []types.FuObject{},
		kwargs: make(map[string]types.FuObject),
	}
	result, errs := fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{}, dirContents("."))

	// easiest case: create a single dir
	args.args = types.MakeFuList("foo")
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo"}, dirContents("."))
	assert.True(t, isDir("foo"))

	// create multiple dirs, including "foo" which already exists
	args.args = types.MakeFuList("meep/meep/meep", "foo", "meep/beep")
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo", "meep"}, dirContents("."))
	assert.True(t, isDir("foo"))
	assert.True(t, isDir("meep/meep"))
	assert.True(t, isDir("meep/meep/meep"))
	assert.True(t, isDir("meep/beep"))

	// now with an error in the middle of the list (*but* we still
	// create the other requested dirs!)
	testutils.TouchFiles("meep/zap")
	args.args = types.MakeFuList("meep/bap", "meep/zap/zip", "foo/bar")
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[0].Error())
	assert.True(t, isDir("meep/bap"))
	assert.True(t, isDir("foo/bar"))

	// finally, with multiple errors
	args.args = append(args.args, types.FuString("meep/zap/blop"))
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[0].Error())
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[1].Error())
}

func Test_remove(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	args := FunctionArgs{
		args:   types.MakeFuList(),
		kwargs: make(map[string]types.FuObject),
	}

	// remove() doesn't care about empty arg list (same reason as mkdir())
	result, errs := fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))

	// remove() ignores non-existent files
	args.args = types.MakeFuList("foo", "bar/bleep/meep", "qux")
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))

	// remove() removes regular files
	testutils.TouchFiles("foo", "bar/bleep/meep", "bar/bleep/feep", "qux")
	args.args = types.MakeFuList("foo", "bar/bleep/meep", "bogus")
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"bar", "qux"}, dirContents("."))
	assert.Equal(t, []string{"bleep"}, dirContents("bar"))
	assert.Equal(t, []string{"feep"}, dirContents("bar/bleep"))

	// remove() removes files and directories too
	testutils.TouchFiles("foo", "bar/bleep/meep", "qux")
	args.args = types.MakeFuList("bogus", "bar", "morebogus", "qux")
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo"}, dirContents("."))

	// remove() fails if it tries to delete from an unwriteable directory
	testutils.TouchFiles("bar/bong", "qux/bip")
	testutils.ChmodRO("bar")
	defer testutils.ChmodOwnerAll("bar")

	args.args = types.MakeFuList("bar", "qux")
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, "remove bar/bong: permission denied", errs[0].Error())
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

func Test_FileNode(t *testing.T) {
	args := FunctionArgs{
		runtime: minimalRuntime(),
		args:    []types.FuObject{types.FuString("a.txt")},
	}
	node0, errs := fn_FileNode(args)
	assert.Equal(t, 0, len(errs))
	node1, errs := fn_FileNode(args)
	assert.Equal(t, 0, len(errs))

	// panic on unexpected type
	_ = node0.(*dag.FileNode)
	_ = node1.(*dag.FileNode)

	assert.Equal(t, "a.txt", node0.(dag.Node).Name())
	assert.True(t, node0.Equal(node1))

	// FileNode is a factory: it will return existing node objects
	// rather than create new ones
	assert.True(t, node0 == node1)
}

func Test_ActionNode(t *testing.T) {
	args := FunctionArgs{
		runtime: minimalRuntime(),
		args:    []types.FuObject{types.FuString("test/x")},
	}
	node0, errs := fn_ActionNode(args)
	assert.Equal(t, 0, len(errs))

	_ = node0.(*dag.ActionNode)
	assert.Equal(t, "test/x:action", node0.ValueString())
	assert.Equal(t, "test/x:action", node0.(dag.Node).Name())

	node1, errs := fn_ActionNode(args)
	assert.True(t, node0 == node1)
}
