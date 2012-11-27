package runtime

import (
	"fubsy/types"
	"fubsy/dag"
)

type BuildRule struct {
	runtime *Runtime
	targets types.FuObject
	sources types.FuObject
	action dag.Action
	locals Namespace
}

func NewBuildRule(
	runtime *Runtime,
	targets types.FuObject,
	sources types.FuObject) *BuildRule {
	return &BuildRule{
		runtime: runtime,
		targets: targets,
		sources: sources,
	}
}
