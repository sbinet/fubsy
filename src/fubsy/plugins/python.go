// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build python

package plugins

import (
	"errors"
	"fmt"
	"strings"
	"unsafe"

	py "github.com/sbinet/go-python"

	"fubsy/log"
	"fubsy/types"
)

// #include <Python.h>
// #include <stdlib.h>
// #include "empython.h"
import "C"

type PythonPlugin struct {
}

func NewPythonPlugin() (MetaPlugin, error) {
	py.Initialize()
	return PythonPlugin{}, nil
}

func (self PythonPlugin) String() string {
	return "PythonPlugin"
}

func (self PythonPlugin) InstallBuiltins(builtins BuiltinList) error {
	for idx := 0; idx < builtins.NumBuiltins(); idx++ {
		_, code := builtins.Builtin(idx)
		fnptr := *(*unsafe.Pointer)(unsafe.Pointer(&code))
		C.set_callback(C.int(idx), fnptr)
	}

	if C.install_builtins() < 0 {
		return errors.New(
			"unknown error setting up Python environment (out of memory?)")
	}
	return nil
}

//export callBuiltin
func callBuiltin(
	pfunc unsafe.Pointer, numargs C.int, cargs unsafe.Pointer) (
	*C.char, *C.char) {

	log.Debug(log.PLUGINS, "callBuiltin: calling Go function at %p", pfunc)
	var fn types.FuCode

	fuargs := make([]types.FuObject, numargs)
	for i := uintptr(0); i < uintptr(numargs); i++ {
		// cargs is really a C char **, i.e. a pointer to an array of
		// char *. argp is a pointer to the i'th member of cargs. This
		// is just C-style array lookup with pointer arithmetic, but
		// in Go syntax.
		argp := unsafe.Pointer(uintptr(cargs) + i*unsafe.Sizeof(cargs))
		arg := C.GoString(*(**C.char)(argp))
		fuargs[i] = types.MakeFuString(arg)
	}
	args := types.MakeBasicArgs(nil, fuargs, nil)

	fn = *(*types.FuCode)(unsafe.Pointer(&pfunc))
	log.Debug(log.PLUGINS, "followed unsafe.Pointer to get %p", fn)
	result, err := fn(args)

	if len(err) > 0 {
		errmsgs := make([]string, len(err))
		for i, err := range err {
			errmsgs[i] = err.Error()
		}
		return nil, C.CString(strings.Join(errmsgs, "\n"))
	}
	var cresult *C.char
	if result != nil {
		cresult = C.CString(result.String())
	}
	return cresult, nil
}

func (self PythonPlugin) Run(content string) (types.ValueMap, error) {
	var ccontent = C.CString(content)
	var cerror *C.char
	var cvalues *C.valuelist_t

	defer C.free(unsafe.Pointer(ccontent))
	if C.run_python(ccontent, &cerror, &cvalues) < 0 {
		return nil, errors.New(C.GoString(cerror))
	}
	defer C.free_valuelist(cvalues)

	valuemap := types.NewValueMap()
	names := uintptr(unsafe.Pointer(cvalues.names))
	values := uintptr(unsafe.Pointer(cvalues.values))
	var cname *C.char
	var name string
	var cvalue *C.PyObject

	for i := uintptr(0); i < uintptr(cvalues.numitems); i++ {
		offset := i * unsafe.Sizeof(i)
		cname = *(**C.char)(unsafe.Pointer(names + offset))
		cvalue = *(**C.PyObject)(unsafe.Pointer(values + offset))
		name = C.GoString(cname)
		valuemap.Assign(name, PythonCallable{name: name, callable: cvalue})
	}
	return valuemap, nil
}

func (self PythonPlugin) Close() {
	// argh, go-python doesn't wrap this
	//py.Py_Finalize()
}

type PythonCallable struct {
	types.NullLookupT
	name     string
	callable *C.PyObject
}

func MakePythonCallable(name string, callable *C.PyObject) PythonCallable {
	return PythonCallable{name: name, callable: callable}
}

func (self PythonCallable) Typename() string {
	return "python function"
}

func (self PythonCallable) String() string {
	return "python:" + self.name + "()"
}

func (self PythonCallable) ValueString() string {
	return self.name
}

func (self PythonCallable) CommandString() string {
	panic("functions should not be interpolated into command strings!")
}

func (self PythonCallable) Equal(other_ types.FuObject) bool {
	other, ok := other_.(PythonCallable)
	return ok && self.callable == other.callable && self.name == other.name
}

func (self PythonCallable) Add(other types.FuObject) (types.FuObject, error) {
	return types.UnsupportedAdd(self, other, "")
}

func (self PythonCallable) List() []types.FuObject {
	return []types.FuObject{self}
}

func (self PythonCallable) ActionExpand(ns types.Namespace, ctx *types.ExpandContext) (
	types.FuObject, error) {
	return self, nil
}

// FuCallable methods

func (self PythonCallable) Name() string {
	return self.name
}

func (self PythonCallable) Code() types.FuCode {
	return func(argsource types.ArgSource) (types.FuObject, []error) {
		return self.callPython(argsource)
	}
}

func (self PythonCallable) CheckArgs(argsource types.ArgSource) error {
	// let Python raise TypeError if it's unhappy
	return nil
}

func (self PythonCallable) callPython(argsource types.ArgSource) (types.FuObject, []error) {
	args := argsource.Args()

	// build a slice of strings, which will then be converted to
	// Python strings in C (this way we only copy the bytes once, at
	// the cost of C code knowing the internals of Go slices and
	// strings)
	sargs := make([]string, len(args))
	for i, arg := range args {
		switch arg.(type) {
		case types.FuString:
			sargs[i] = arg.ValueString()
		default:
			err := fmt.Errorf(
				"bad argument type: all arguments must be strings, "+
					"but argument %d is %s %s",
				i+1, arg.Typename(), arg.String())
			return nil, []error{err}
		}
	}

	var cerror *C.char
	var cresult *C.char
	var result types.FuObject
	cresult = C.call_python(self.callable, unsafe.Pointer(&sargs), &cerror)

	if cerror != nil {
		err := C.GoString(cerror)
		return nil, []error{errors.New(err)}
	}
	if cresult != nil {
		result = types.MakeFuString(C.GoString(cresult))
	}

	return result, nil
}
