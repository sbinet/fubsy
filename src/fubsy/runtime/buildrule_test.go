// Copyright © 2013, Greg Ward. All rights reserved.
// Use of this source code is governed by a BSD-style license that can
// be found in the LICENSE.txt file.

package runtime

import (
	"testing"

	"github.com/stretchrcom/testify/assert"

	"fubsy/dag"
	"fubsy/types"
)

func Test_BuildRule_setLocals(t *testing.T) {
	targets := []dag.Node{dag.NewStubNode("foo")}
	sources := []dag.Node{dag.NewStubNode("bar"), dag.NewStubNode("qux")}
	ns := types.NewValueMap()
	rule := NewBuildRule(nil, targets, sources)

	rule.setLocals(ns)
	var val types.FuObject
	var ok bool

	val, ok = ns.Lookup("whatever")
	assert.False(t, ok)
	val, ok = ns.Lookup("target")
	assert.False(t, ok)
	val, ok = ns.Lookup("targets")
	assert.False(t, ok)

	val, ok = ns.Lookup("TARGET")
	assert.True(t, ok)
	assert.Equal(t, "foo", val.ValueString())
	assert.Equal(t, "foo", val.(*dag.StubNode).Name())

	val, ok = ns.Lookup("SOURCE")
	assert.True(t, ok)
	assert.Equal(t, "bar", val.ValueString())
	assert.Equal(t, "bar", val.(*dag.StubNode).Name())

	val, ok = ns.Lookup("TARGETS")
	assert.True(t, ok)
	assert.Equal(t, 1, len(val.List()))
	assert.Equal(t, `["foo"]`, val.String())

	val, ok = ns.Lookup("SOURCES")
	assert.True(t, ok)
	assert.Equal(t, 2, len(val.List()))
	assert.Equal(t, `["bar", "qux"]`, val.String())
}
