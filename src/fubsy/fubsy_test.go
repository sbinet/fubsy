// Copyright © 2012-2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package main

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/testutils"
)

func Test_findScripts(t *testing.T) {
	cleanup := testutils.Chtemp()
	defer cleanup()

	var script string
	var err error

	// case 1: user specifies the script to run, regardless of whether
	// it exists or not
	script, err = findScript("foo")
	assert.Equal(t, "foo", script)
	assert.Nil(t, err)

	// case 2: no *.fubsy files in current dir
	script, err = findScript("")
	assert.Equal(t,
		"main.fubsy not found (and no other *.fubsy files found)",
		err.Error())

	// case 3: only main.fubsy exists
	testutils.TouchFiles("main.fubsy")
	script, err = findScript("")
	assert.Equal(t, "main.fubsy", script)
	assert.Nil(t, err)

	// case 4: multiple *.fubsy files exist, including main.fubsy
	testutils.TouchFiles("a.fubsy", "b.fubsy")
	script, err = findScript("")
	assert.Equal(t, "main.fubsy", script)
	assert.Nil(t, err)

	// case 5: multiple *.fubsy files exist, not including main.fubsy
	remove("main.fubsy")
	script, err = findScript("")
	assert.Equal(t, "", script)
	assert.True(t,
		strings.HasPrefix(
			err.Error(),
			"main.fubsy not found, and multiple *.fubsy files exist",
		))

	// case 6: exactly one *.fubsy file exists, and it's not main.fubsy
	remove("a.fubsy")
	script, err = findScript("")
	assert.Equal(t, "b.fubsy", script)
	assert.Nil(t, err)
}

func remove(name string) {
	err := os.Remove(name)
	if err != nil {
		panic(err)
	}
}
