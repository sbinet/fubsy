How it works
============

Now that we've seen four small but realistic examples, this is a good
time to delve into how Fubsy really works: what exactly is going on
behind the scenes in these build scripts?

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

(I've dropped the ``source`` and ``headers`` variables here to make
things more explicit.)

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
no need to re-run it unless something that it depends on changed. So
it's useful to have a node in your dependency graph that records the
successful execution of a unit test. Static analysis tools are
similar.)

Here's how it works. First, Fubsy figures out what targets to build.
By default, its goal is to build all *final targets*, i.e. target
files that are not themselves the source for some later target. You
can specify which targets to build on the command line, e.g. ::

    fubsy myapp.o

if you're having trouble compiling ``myapp.c`` and just want to
concetrate on it for the moment.

(Incidentally, *source* and *target* are relative terms: ``myapp.o``
is a target derived from ``myapp.c``, but a source for ``myapp``. Any
file that is both a source and a target is also called an
*intermediate target*. Any file that is not built from something else
is called an *original source*. Original sources are what you modify
and keep in source control; everything else is temporary and
disposable.)

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

Once Fubsy has determined the targets that it's trying to build -- the
goal nodes -- it walks the dependency graph to find all *relevant*
nodes, i.e. all ancestors of the goal nodes. The end points of this
walk are the original source nodes that the goal nodes depend on.
Fubsy examines each relevant original source node to determine which
have changed since the last build; if there is no information for a
particular node, e.g. because there was no previous build, it is
considered to have changed. The relevant children of those modified
nodes are the *initial rebuild set*, the set of nodes that must be
rebuilt.

Having computed the initial rebuild set, Fubsy starts the build
proper. It rebuilds each node in the rebuild set by executing the
actions associated with it. Then it checks if the node was actually
changed by the rebuild, which can usefully short-circuit builds after
a whitespace-only change in source code (which does not usually affect
compiler output). If the node was indeed changed, Fubsy adds all of
its relevant children to the rebuild set and continues building until
the rebuild set is empty.

Example: initial build
----------------------

An example should clarify things. Let's continue with the case above,
building ``tool1`` and ``tool2``. By default, the goal consists of all
final targets. To make things interesting, let's suppose you specify a
different goal: ``fubsy tool2``, which means the relevant nodes are a
subset of the whole graph:

  [diagram: same as above, with tool2 marked as the goal node,
  and tool2.o, tool2.c, util.o, util.c, util.h as relevant nodes]

On the first build, Fubsy has no record of what came before, so it
considers that all of the relevant original source nodes are changed,
which implies the initial rebuild set too:

  [diagram: same as above, with tool2.c, util.c, util.h marked
  "changed" and tool2.o, util.o marked "stale"]

(A node in state "stale" is in the rebuild set.) So Fubsy has to build
the two object files::

    cc -o tool2.o tool2.c
    cc -o util.o util.c

After building each node, Fubsy checks if it has changed. Again, since
this is the first build and we have no previous information, it
considers each to have changed, which means the graph now looks like this:

  [diagram: same as above, but now tool2.o, util.o are "built" and "changed"
  and tool2 is "stale"]

Build all nodes in the rebuild set::

    cc -o tool2 tool2.o util.o

and we're done, because the rebuild set is now empty:

  [diagram: same as above, but now tool2 is "built"]

Example: incremental rebuild
----------------------------

Of course, if all you want to do is build everything, you don't need a
fancy build tool like Fubsy. A shell script will work just fine. The
real value of Fubsy becomes apparent when you modify your source code.
To make things interesting, let's say we've made a real change in
``tool2.c``, i.e. one that affects the object code. Again, we'll
assume the goal node is just ``tool2``. So after Fubsy determines
relevant nodes and the initial rebuild set, we have this:

  [diagram: as above, with tool2 the goal node, same relevant nodes;
  tool2.c "changed"; tool2.o "stale"]

The first pass over the rebuild set::

    cc -o tool2.o tool2.c

updates the graph to

  [diagram: as above, but now tool2.o is "built", "changed"]

which requires one more pass to empty the rebuild set::

  cc -o tool2 tool2.o util.o

Because none of the ancestors of ``util.o`` changed, we didn't have to
rebuild it, and used the pre-existing version of ``util.o`` to link
``tool2``.

Example: short-circuit rebuild
------------------------------

Now let's say you edit a comment in ``util.h``. Assuming this does not
affect the object code, this should avoid unnecessary rebuilds apart
from some object files: a short-circuit rebuild. First, Fubsy
determines the relevant nodes, original source nodes, and initial
rebuild set:

  [diagram: tool2 is the goal, relevant set is the same, util.h is
  "changed", util.o, tool2.o are "stale"]

Because both ``util.o`` and ``tool2.o`` depend on (are children of)
``util.h``, we must rebuild both. Fubsy has no idea that you only
changed a comment, so it has no way to know that your change is
trivial until it rebuilds the children of ``util.h``::

    cc -o tool2.o tool2.c
    cc -o util.o util.c

After rebuilding each object file, Fubsy examines it and determines
that it is in fact unchanged since the last build:

  [diagram: util.o, tool2.o are "unchanged", "built"]

Because both are unchanged, Fubsy adds nothing to the rebuild set,
which is now empty. So the build is done without the expense of
unnecessarily relinking ``tool2``.
