package runtime

type BuildRule struct {
	runtime *Runtime
	targets FuObject
	sources FuObject
	locals Namespace
}

func NewBuildRule(runtime *Runtime, targets FuObject, sources FuObject) *BuildRule {
	return &BuildRule{
		runtime: runtime,
		targets: targets,
		sources: sources,
	}
}
