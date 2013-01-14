How it works
============

Now that we've seen four small but realistic examples, this is a good
time to delve into how Fubsy really works: what exactly is going on
behind the scenes in these build scripts?

.. note:: This section is mostly accurate as of Fubsy 0.0.2.

The dependency graph
--------------------

The central data structure that drives everything in Fubsy is the
*dependency graph*, which describes how your source and target files
are related. The purpose of the *main* phase of a Fubsy build script
is to construct the dependency graph by describing the relationships
between source and target files. This is your job, hopefully aided by
plugins like ``c`` or ``java``.

Let's revisit our naive build script for ``myapp.c`` and friends. The
core of that script was a single build rule::

    "myapp": ["myapp.c", "util.c", "util.h"] {
        "cc -o $TARGET myapp.c util.c"
    }

(I've dropped all variables and filefinders here to make things more
explicit.)

This build rule specifies a simple dependency graph:

    [diagram: myapp depends on myapp.c, util.c, util.h]

The list of actions ("cc ...") is attached to the target file(s). That
is, Fubsy knows exactly one way to build every target file in a
particular invocation. (The actions can change across runs, e.g. if
you override the ``CFLAGS`` build variable on the command line.) This
graph makes visible the weakness of the naive build script: changing
*any* source file means recompiling *all* of them.

The smarter build script for ``myapp`` invokes the ``c`` plugin::

    c.binary("myapp", ["myapp.c", "util.c"])

which effectively adds several build rules::

    "myapp": ["myapp.o", "util.o"] {
        "$LINKER -o $TARGET $SOURCES"
    }
    "myapp.o": ["myapp.c"] {
        "$CC -o $TARGET $SOURCES"
    }
    "util.o": ["util.c"] {
        "$CC -o $TARGET $SOURCES"
    }
    depends("myapp.o", "util.h")
    depends("util.o", "util.h")

You can see here the two types of dependencies: *direct* dependencies
spelled out directly in the build rule, like "myapp.o depends on
myapp.c"; and *indirect* dependencies added with a call to
``depends()`` outside of the build rule, like "myapp.o depends on
util.h". The reason for the distinction is that we don't want
``util.h`` to be included in ``$SOURCES`` in the build rule, but we do
want it to affect Fubsy's decisions about what to rebuild.

The dependency graph resulting from this build script is more complex,
but will result in a much more scalable build:

  [diagram: myapp depends on 2 .o files, which depend on .c and .h]

Everything you do in the *main* phase of your build script is there to
construct the dependency graph. Fubsy then uses that data in the
*build* phase.

The build phase
---------------

You'll notice that we haven't included an explicit *build* phase,
like ::

   build {
       ...
   }

in any of our sample scripts. That's because the *build* phase is
where Fubsy takes over and conditionally executes the actions it finds
in your dependency graph based on the state of the nodes in the graph
(source and target files in your working directory).

(At this point, I'm going to stop talking about files and talk about
nodes in the dependency graph instead. Nodes are *usually* files, but
can be any resource involved in a dependency relationship. For
example, unit tests typically don't generate any output: they run and
either pass or fail. If a unit test passed in the last build, there's
no need to re-run it unless something that it depends on has changed.
So it's useful to have a node in your dependency graph that records
the successful execution of a unit test. Similar reasoning applies to
static analysis tools.)

Here's how it works. First, Fubsy figures out the set of *goal nodes*,
i.e. which targets to build. By default, the goal is the set of all
*final targets*: targets that are not themselves the source for some
later target. Alternately, you can specify which targets to build on
the command line--e.g., if you're having trouble compiling ``myapp.c``
and just want to concentrate on it for the moment::

    fubsy myapp.o

(Incidentally, *source* and *target* are relative terms: ``myapp.o``
is a target derived from ``myapp.c``, but a source for ``myapp``. Any
node that is both a source and a target is also called an
*intermediate target*. Any node that is not built from something else
is called an *original source*. Original sources are what you modify
and keep in source control; everything else is temporary and
disposable. And final targets, as already described, are nodes that
are not the source to any other node--typically deployable
executables, packages, or installers.)

Let's cook up a slightly more complex example to illustrate: now we're
going to build two binaries, ``tool1`` and ``tool2``, from the
following source files::

    tool1.c
    tool2.c
    util.c + util.h
    misc.c + misc.h

``tool1`` depends on both ``util.c`` and ``misc.c``, but ``tool2``
depends only on ``util.c``. Here is the build script::

    import c

    main {
        c.binary("tool1", ["tool1.c", "util.c", "misc.c"])
        c.binary("tool2", ["tool2.c", "util.c"])
    }

And here is the dependency graph described by that build script:

  [diagram:
  tool1 -> tool1.o -> tool1.c, util.h, misc.h
  tool1 -> util.o -> util.c, util.h
  tool1 -> misc.o -> misc.c, misc.h
  tool2 -> tool2.o -> tool2.c, util.h
  tool2 -> util.o -> util.c, util.h
  ]

Once Fubsy has determined the targets that it's trying to build--the
goal nodes--it constructs a second dependency graph containing only
the goal nodes and their ancestors. This step is also used to expand
any filefinder nodes that have survived this far: e.g. if there is a
node like ``<src/**/*.java>`` in the graph, it is replaced with nodes
for every matching file. We'll call this second graph the *build
graph*.

Then, Fubsy walks the new dependency graph in *topological order*:
that is, if node *B* depends on (is a child of) node *A*, it will
visit *A* before visiting *B*. In fact, it will visit all nodes that
*B* depends on before visiting *B*. As it visits each node, Fubsy
performs the following steps:

  #. if the node is an original source node (it depends on nothing
     else), skip to the next node in topological order
  #. if the node is *tainted* because one of its ancestors failed to
     build, skip to the next node
  #. if the node is missing or *stale* (one of its parents has changed
     since the last build), build it

Once those three tests have been applied to every node in the goal
set, then the build is finished. If there were any failures, the whole
build is a failure.

Example: initial build
----------------------

An example should clarify things. Let's continue with the case above,
building ``tool1`` and ``tool2``. By default, the goal consists of all
final targets. To make things interesting, let's suppose you specify a
different goal: ``fubsy tool2``, which means the build graph contains only ancestors of ``tool2``:

  [diagram: same as above, with non-ancestors of tool2 removed]

Let's assume that Fubsy's topological graph walk visits all of the
original source nodes first.

  [diagram: same as above, with tool2.c, util.c, util.h "skipped"]

When it visits ``tool2.o``, Fubsy looks in the filesystem and sees
that that node is missing, so builds it::

    cc -o tool2.o tool2.c

Now the graph looks like this:

  [diagram: same as above, with tool2.o marked "built"]

Next in line is ``util.o``, which is also missing::

    cc -o util.o util.c

Finally we visit and build ``tool2``::

    cc -o tool2 tool2.o util.o

We're done; every node in the graph has been visited:

  [diagram: same as above, but now util.o and tool2 are "built"]

Example: incremental rebuild
----------------------------

Of course, if all you want to do is build everything, you don't need a
fancy build tool like Fubsy. A shell script will work just fine. The
real value of Fubsy becomes apparent when you modify your source code.
To make things interesting, let's say we've made a real change in
``tool.c``, i.e. one that affects the object code. Again, we'll
assume the goal node is just ``tool2``.

The initial build graph is the same as in the previous example, and
the first couple of steps are the same. Things change slightly when
Fubsy reaches ``tool2.o``: this time the target node exists, but one
of its parents (``tool2.c``) has changed since the last build. So
Fubsy has to rebuild the target::

    cc -o tool2.o tool2.c

The graph looks the same as it did at this point in the previous example:

  [diagram: as above, tool.o marked "built"]

Next we visit ``util.o``. But none of its parents have changed, so no
rebuild is required.

  [diagram: as above, util.o marked "skipped"]

Finally we visit the ``tool2`` node. One of its parents, ``tool2.o``,
has changed, so we have to rebuild the final target::

    cc -o tool2 tool2.o util.o

Because none of the ancestors of ``util.o`` changed, we didn't have to
rebuild it, and used the pre-existing version of ``util.o`` to link
``tool2``.

Example: short-circuit rebuild
------------------------------

Now let's say you edit a comment in ``util.h``. Assuming this does not
affect the object code, this should avoid unnecessary downstream
rebuilds: a short-circuit rebuild.

When Fubsy reaches ``tool2.o``, it will inspect its parents and
realize that ``util.h`` has changed; likewise for ``util.o``. So those
two files must be rebuilt::

    cc -o tool2.o tool2.c
    cc -o util.o util.c

But because you only changed a comment, the object code in both files
is unchanged. So when Fubsy visits ``myapp``, none of that node's
parents are changed, and it can skip rebuilding. The final graph:

  [diagram: as above, with tool2.o, util.o "built" and tool2 "skipped"]

We've saved the cost of linking one binary. In this trivial example,
that's not much. But it can make a difference in larger builds, and
Fubsy is designed to scale up to very large builds indeed.
