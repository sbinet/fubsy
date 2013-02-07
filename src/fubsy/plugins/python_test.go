// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

// +build python

package plugins

import (
	"testing"

	"github.com/stretchrcom/testify/assert"
)

func Test_PythonPlugin_Run(t *testing.T) {
	pp, err := NewPythonPlugin()
	assert.Nil(t, err)
	values, err := pp.Run(`
foo = ["abc", "def"]
bar = "!".join(foo)`)

	// PythonPlugin doesn't yet harvest Python values, so we cannot do
	// anything to test values
	_ = values
	assert.Nil(t, err)

	values, err = pp.Run("foo = 1/0")
	assert.Equal(t, "inline Python plugin raised an exception", err.Error())
}
