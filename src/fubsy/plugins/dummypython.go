// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build !python

package plugins

import (
	"fubsy/types"
)

// Dummy version of PythonPlugin, used when the build host does not
// have Python.h etc.

type PythonPlugin struct {
}

func NewPythonPlugin() (MetaPlugin, error) {
	return nil, NotAvailableError{"Python"}
}

func (self PythonPlugin) Run(content string) (types.ValueMap, error) {
	panic("dummy implementation")
}

func (self PythonPlugin) Close() {
	panic("dummy implementation")
}
