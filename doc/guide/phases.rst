Phases
======

I mentioned above that a Fubsy build script can contain multiple
phases, corresponding to the phases of Fubsy's own execution. Those
phases are:

  *options*
    add command-line options and variables that the user can pass
    to ``fubsy`` (executes very early, so Fubsy can tell if the
    command line is valid without churning through lots of
    expensive dependency analysis)

  *configure*
    examine the build system to figure out which compilers, tools,
    libraries, etc. are present in order to influence later phases

  *main*
    specifies the resources (files) involved in your build, constructing
    the graph of dependencies that will drive everything

  *build*
    follow the graph of dependencies to rebuild out-of-date files
    (i.e. conditionally execute actions based on dependencies between
    related resources)

  *clean*
    remove some or all build products (typically used in a separate
    invocation of ``fubsy``: running *build* and *clean* in the same
    invocation would be pointless)

All phases except ``main`` are optional; a build script with no
``main`` phase would have an empty dependency graph, so nothing to
build.

Currently, Fubsy always runs the *main* phase to define the graph of
dependencies, followed by the *build* phase to walk the graph and
build stale or missing targets. When *options* is implemented, it will
always run first, since *main* will depend on user-defined
command-line options and variables.

The other phases depend on user actions. For example, the *clean*
phase will run if and only if the user executes ::

    fubsy clean

The *configure* phase will run if the user executes ::

    fubsy configure

However, there will probably be circumstances under which Fubsy runs
*configure* automatically, e.g. in a fresh working dir that has never
been configured. This is all to be sorted out in the future.

.. note:: So far, only *main* and *build* are implemented. The *main*
          phase must be explicitly provided in every build script, and
          the *build* phase is implicit. It's unclear what it would
          mean if a build script provided an explicit *build* phase.
          It's entirely possible that using the same mechanism to
          describe both explicitly coded phases like *main* and the
          implicit, behind-the-scenes *build* phase is a bad idea.
