// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build python

package plugins

import (
	"errors"
	//"fmt"
	"strings"
	"unsafe"

	py "github.com/sbinet/go-python"

	"fubsy/log"
	"fubsy/types"
)

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
		C.setCallback(C.int(idx), fnptr)
	}

	if C.installBuiltins() < 0 {
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

func (self PythonPlugin) Run(content string) (
	types.ValueMap, error) {

	result := py.PyRun_SimpleString(content)
	if result < 0 {
		// there's no way to get the traceback info... but it doesn't
		// really matter, since Python prints the traceback to stderr
		return nil, errors.New("inline Python plugin raised an exception")
	}
	return nil, nil
}

func (self PythonPlugin) Close() {
	// argh, go-python doesn't wrap this
	//py.Py_Finalize()
}
