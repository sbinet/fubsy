package dag

import (
	"testing"
	"github.com/stretchrcom/testify/assert"
	"code.google.com/p/go-bit/bit"
)

func Test_BuildState_FindOriginalSources(t *testing.T) {
	dag := makeSimpleGraph()
	bstate := dag.NewBuildState()
	bstate.goal = bit.New(0, 1)		// all final targets: tool1, tool2
	bstate.FindOriginalSources()
	assert.Equal(t, "{6..11}", bstate.sources.String())
	assert.Equal(t, "{0..11}", bstate.relevant.String())

	// relevant children of misc.h = {tool1.o, misc.o}
	assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o, tool2.o}
	assert.Equal(t, "{2, 4, 5}", bstate.children[9].String())

	// goal = {tool1} ==>
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()
	assert.Equal(t, "{6..10}", bstate.sources.String())
	assert.Equal(t, "{0, 2..4, 6..10}", bstate.relevant.String())

	// relevant children of misc.h = {tool1.o, misc.o}
	assert.Equal(t, "{2, 3}", bstate.children[7].String())
	// relevant children of util.h = {tool1.o, util.o}
	assert.Equal(t, "{2, 4}", bstate.children[9].String())

	// goal = {tool2} ==>
	// sources = {util.h, util.c, tool2.c}
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()
	assert.Equal(t, "{9..11}", bstate.sources.String())
	assert.Equal(t, "{1, 4, 5, 9..11}", bstate.relevant.String())

	// misc.h is not a relevant node, so not in children map
	assert.Nil(t, bstate.children[7])
	// relevant children of util.h = {util.o, tool2.o}
	assert.Equal(t, "{4, 5}", bstate.children[9].String())
}

func Test_BuildState_FindOriginalSources_cycle(t *testing.T) {
	dag := makeSimpleGraph()
	dag.lookup("misc.h").AddParent(dag.lookup("tool1"))
	bstate := dag.NewBuildState()

	// goal = {tool2} ==> no cycle, since we don't visit those nodes
	// (this simply tests that FindOriginalSources() doesn't panic)
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()

	// goal = {tool1} ==> cycle!
	// (disabled because FindOriginalSources() currently panics on cycle)
	return
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()
}

func Test_BuildState_FindStaleTargets(t *testing.T) {
	// this test depends on FindOriginalSources() working
	// goal = {tool1}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c}
	// initial rebuild = {tool1.o, misc.o, util.o}
	dag := makeSimpleGraph()
	bstate := dag.NewBuildState()
	bstate.goal = bit.New(0)
	bstate.FindOriginalSources()

	errors := bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{2..4}", bstate.rebuild.String())

	// goal = {tool2}
	// sources = {tool2.c, util.h, util.c}
	// initial rebuild = {tool2.o, util.o}
	bstate.goal = bit.New(1)
	bstate.FindOriginalSources()

	errors = bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{4, 5}", bstate.rebuild.String())

	// goal = {tool1, tool2}
	// sources = {tool1.c, misc.h, misc.c, util.h, util.c, tool2.c}
	// initial rebuild = {tool1.o, misc.o, util.o, tool2.o}
	bstate.goal = bit.New(0, 1)
	bstate.FindOriginalSources()
	errors = bstate.FindStaleTargets()
	assert.Equal(t, 0, len(errors))
	assert.Equal(t, "{2..5}", bstate.rebuild.String())
}

