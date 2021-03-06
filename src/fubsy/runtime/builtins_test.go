// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"errors"
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

func Test_BuiltinList(t *testing.T) {
	blist := BuiltinList{}
	fn, ok := blist.Lookup("foo")
	assert.False(t, ok)
	assert.Nil(t, fn)

	callable := types.NewFixedFunction("foo", 3, nil)
	blist.builtins = append(blist.builtins, callable)
	fn, ok = blist.Lookup("foo")
	assert.True(t, ok)
	assert.Equal(t, callable, fn)

	blist.builtins = append(
		blist.builtins, types.NewFixedFunction("bar", 0, nil))
	blist.builtins = append(
		blist.builtins, types.NewFixedFunction("bop", 0, nil))
	blist.builtins = append(
		blist.builtins, types.NewFixedFunction("bam", 0, nil))
	blist.builtins = append(
		blist.builtins, types.NewFixedFunction("pow", 0, nil))

	assert.Equal(t, 5, blist.NumBuiltins())
	actual := make([]string, 0, 5)
	visit := func(name string, code types.FuObject) error {
		actual = append(actual, name)
		if name == "bam" {
			return errors.New("bam!")
		}
		return nil
	}
	err := blist.ForEach(visit)
	assert.Equal(t, []string{"foo", "bar", "bop", "bam"}, actual)
	assert.Equal(t, "bam!", err.Error())
}

func Test_defineBuiltins(t *testing.T) {
	ns := defineBuiltins()

	fn, ok := ns.Lookup("println")
	assert.True(t, ok)
	assert.NotNil(t, fn)
	assert.Equal(t, fn.(types.FuCallable).Code(), types.FuCode(fn_println))

	fn, ok = ns.Lookup("remove")
	assert.True(t, ok)
	assert.NotNil(t, fn)
	assert.Equal(t, fn.(types.FuCallable).Code(), types.FuCode(fn_remove))

	// there will never be a builtin with this name: guaranteed!
	fn, ok = ns.Lookup("sad425.-afgasdf")
	assert.False(t, ok)
	assert.Nil(t, fn)
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

	args := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(nil, []types.FuObject{}, nil),
	}

	result, errs := fn_println(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	data, err := ioutil.ReadFile("stdout")
	assert.Nil(t, err)
	assert.Equal(t, "\n", string(data))
	rfile.Truncate(0)
	rfile.Seek(0, 0)

	args.SetArgs(types.MakeStringList("hello", "world").List())
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
	pargs := []types.FuObject{}
	args := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(nil, pargs, nil),
	}
	result, errs := fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{}, dirContents("."))

	// easiest case: create a single dir
	pargs = types.MakeStringList("foo").List()
	args.SetArgs(pargs)
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo"}, dirContents("."))
	assert.True(t, isDir("foo"))

	// create multiple dirs, including "foo" which already exists
	pargs = types.MakeStringList("meep/meep/meep", "foo", "meep/beep").List()
	args.SetArgs(pargs)
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
	pargs = types.MakeStringList("meep/bap", "meep/zap/zip", "foo/bar").List()
	args.SetArgs(pargs)
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 1, len(errs))
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[0].Error())
	assert.True(t, isDir("meep/bap"))
	assert.True(t, isDir("foo/bar"))

	// finally, with multiple errors
	pargs = append(pargs, types.MakeFuString("meep/zap/blop"))
	args.SetArgs(pargs)
	result, errs = fn_mkdir(args)
	assert.Nil(t, result)
	assert.Equal(t, 2, len(errs))
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[0].Error())
	assert.Equal(t, "mkdir meep/zap: not a directory", errs[1].Error())
}

func Test_remove(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	args := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(nil, []types.FuObject{}, nil),
	}

	// remove() doesn't care about empty arg list (same reason as mkdir())
	result, errs := fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))

	// remove() ignores non-existent files
	args.SetArgs(types.MakeStringList("foo", "bar/bleep/meep", "qux").List())
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))

	// remove() removes regular files
	testutils.TouchFiles("foo", "bar/bleep/meep", "bar/bleep/feep", "qux")
	args.SetArgs(types.MakeStringList("foo", "bar/bleep/meep", "bogus").List())
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"bar", "qux"}, dirContents("."))
	assert.Equal(t, []string{"bleep"}, dirContents("bar"))
	assert.Equal(t, []string{"feep"}, dirContents("bar/bleep"))

	// remove() removes files and directories too
	testutils.TouchFiles("foo", "bar/bleep/meep", "qux")
	args.SetArgs(types.MakeStringList("bogus", "bar", "morebogus", "qux").List())
	result, errs = fn_remove(args)
	assert.Nil(t, result)
	assert.Equal(t, 0, len(errs))
	assert.Equal(t, []string{"foo"}, dirContents("."))

	// remove() fails if it tries to delete from an unwriteable directory
	testutils.TouchFiles("bar/bong", "qux/bip")
	testutils.ChmodRO("bar")
	defer testutils.ChmodOwnerAll("bar")

	args.SetArgs(types.MakeStringList("bar", "qux").List())
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
	args := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(nil, types.MakeStringList("a.txt").List(), nil),
		runtime:   minimalRuntime(),
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
	args := RuntimeArgs{
		BasicArgs: types.MakeBasicArgs(nil, types.MakeStringList("test/x").List(), nil),
		runtime:   minimalRuntime(),
	}
	node0, errs := fn_ActionNode(args)
	assert.Equal(t, 0, len(errs))

	_ = node0.(*dag.ActionNode)
	assert.Equal(t, "test/x:action", node0.ValueString())
	assert.Equal(t, "test/x:action", node0.(dag.Node).Name())

	node1, errs := fn_ActionNode(args)
	assert.True(t, node0 == node1)
}
