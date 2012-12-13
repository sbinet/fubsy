Phases
======

I mentioned above that a Fubsy build script can contain multiple
phases, corresponding to the phases of Fubsy's own execution. Those
phases are:

  options
    add command-line options and variables that the user can pass
    to ``fubsy`` (executes very early, so Fubsy can tell if the
    command line is valid without churning through lots of
    expensive dependency analysis)

  configure
    examine the build system to figure out which compilers, tools,
    libraries, etc. are present in order to influence later phases

  main
    specifies the resources (files) involved in your build, constructing
    the graph of dependencies that will drive everything

  build
    follow the graph of dependencies to rebuild out-of-date files
    (i.e. conditionally execute actions based on dependencies between
    related resources)

  clean
    remove some or all build products (typically used in a separate
    invocation of ``fubsy``: running *build* and *clean* in the same
    invocation would be pointless)

All phases except ``main`` are optional; a build script with no
``main`` phase would have an empty dependency graph, so nothing to
build.