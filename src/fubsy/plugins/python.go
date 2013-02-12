// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build python

package plugins

import (
	"errors"

	py "github.com/sbinet/go-python"

	"fubsy/types"
)

type PythonPlugin struct {
}

func NewPythonPlugin() (MetaPlugin, error) {
	py.Initialize()
	return PythonPlugin{}, nil
}

func (self PythonPlugin) Run(content string) (types.ValueMap, error) {
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
