package runtime

import (
	"fubsy/types"
)

type BuildRule struct {
	runtime *Runtime
	targets types.FuObject
	sources types.FuObject
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
