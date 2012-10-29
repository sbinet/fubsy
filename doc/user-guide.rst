===============================
Fubsy: the universal build tool
===============================
----------
User Guide
----------

.. contents::

Introduction
============

Fubsy is a tool for efficiently building software. Roughly speaking:
Fubsy lets you (re)build target files from source files with minimal
effort based on which source files have changed. More generically,
Fubsy is an engine for the *conditional execution* of *actions* based
on the *dependencies* between a collection of related *resources*.

Let's unpack that generic description and see how it relates to a
concrete example. Typically, resources are files: source code that you
maintain plus output files created by compilers, linkers, packagers,
etc. For example, consider a simple C project that consists of four
source files::

    mytool.c
    util.c
    util.h
    README.txt

Initially, your goal is simply to build the executable ``mytool`` by
compiling ``mytool.c`` to ``mytool.o`` and ``util.c`` to ``util.o``,
then linking the two object files together [1]_. More importantly, you
want to perform the minimum necessary work whenever source files
change: if you modify ``mytool.c``, then recompile ``mytool.o`` and
relink the executable. But if you modify ``util.h``, you may need to
recompile both ``util.o`` and ``main.o`` before relinking. This is
exactly the sort of problem that Fubsy is designed for.

.. [1] On Windows, of course, the output files are ``mytool.obj``,
   ``util.obj``, and ``mytool.exe``. Fubsy's core knows nothing of
   this, but its C and C++ plugins take care of these details for you.

Similar tools
-------------

Of course, Fubsy is hardly the first piece of software that attempts
to tackle this problem. Every C programmer is familiar with Make,
which does a reasonable job for small-to-medium C/C++ projects on
Unix-like systems... except for the matter of header file dependencies
(``mytool.o`` depends on ``util.h``), which Fubsy -- or rather,
Fubsy's C plugin -- handles nicely. However, Make has awkward syntax,
poor extensibility, and confusing semantics, which have led many
people over the years to paper over its difficulties by writing
Makefile generators and the like.

Similarly, most Java programmers are familiar with Ant, which attempts
to solve the problem in a radically different way. Ant doesn't provide
much in the way of dependency management (which is surprisingly
difficult to do with Java), but it is extensible in a real programming
language (Java). As a result, it works the same across platforms,
which is more than you can say for Make. Unfortunately, Ant takes
"awkward syntax" to a whole new level by using XML rather than a
custom language, and it is limited to the Java ecosystem, making it
useless for programmers outside that universe.

Some C/C++ programmers are familiar with SCons, which brought a new
level of rigour, consistency, and extensibility to build tools. SCons
puts the graph of dependencies front and centre. It requires
developers to get their dependencies right, guaranteeing a correct
build in exchange for the effort. Additionally, SCons ships with
excellent support for C and C++ which makes many build scripts
trivial. Unfortunately, SCons suffers from poor performance, and its
dependency engine is incapable of handling weird languages like Java
where target filenames are not easily predicted from source filenames.

Fubsy learns from the lessons of the past, finally delivering the
build tool you've wanted all along. Like Make, Fubsy has a simple
custom language designed specifically for writing build scripts, which
makes most build scripts quite concise. Unlike Make, Fubsy uses a
familiar syntax, has local variables, and distinguishes strings from
lists. Like Ant, Fubsy has a small core with most interesting stuff
happening in plugins. Unlike Ant, plugins are trivial to implement:
you can write small "inline" plugins right in your build script for
simple cases, and you can extend Fubsy in any high-level language that
it supports: e.g. Python, Lua, Ruby, JavaScript, ... as long as
someone has implemented a Fubsy "meta-plugin" for a given language,
you can implement plugins in that language. Finally, like SCons, Fubsy
puts the graph of dependencies in the foreground. But unlike SCons,
Fubsy has minimal runtime overhead, and allows you to modify the graph
of dependencies even while the build is running.

A simple C example
==================

Enough with the vague promises; let's see some code.

C the naive way
---------------

Here is a naive build script for that simple C project above::

    # the WRONG WAY to build a C program; in reality, you should
    # use the 'c' plugin!
    main {
        headers = <*.h>
        source = <*.c>

        # rebuild mytool when any source or header file changes
        "mytool": headers + source {
            "cc -o $TARGET $source"
        }
    }

The first thing you notice is that all the code is in the *main*
phase. A Fubsy script can contain multiple phases, corresponding to
different phases in Fubsy's execution. The only phase that must be
present in every build script is *main*, whose purpose is to
describe the graph of dependencies that drives everything Fubsy does.
We'll see the other phases in a little while.

Next we see two variable assignments::

    headers = <*.h>
    source = <*.c>

Since these are inside a phase, they are local to that phase.
(Variable assignments at top level define global variables, visible
throughout the entire build script. By convention, global variables
are UPPERCASE, and local variables are lowercase.)

Since finding files is very common in build scripts, Fubsy has special
syntax for it: angle brackets `<>` contain a space-separated list of
wildcards. (The wildcard syntax is the same as Ant's, e.g.
``<**/*.c>`` finds all ``*.c`` files in your tree, including the top
directory.)

Fubsy wildcards are evaluated as late as possible. At this point,
``headers`` simply contains a reference to a file-finding object that
will expand ``*.h`` when needed. Also, wildcard expansion uses both
the filesystem and the dependency graph. If you have a build rule
somewhere in your script that generates a new ``*.h`` file, the
expansion of ``<*.h>`` will include it.

Finally, the whole point of a build tool is to build something, which
you do in Fubsy with *build rules* like ::

    "mytool": headers + source {
        "cc -o $TARGET $source"
    }

The generic syntax for a build rule is ::

    TARGETS : SOURCES {
        ACTIONS
    }

which means that ``TARGETS`` depend on ``SOURCES``, and can be rebuilt
by executing ``ACTIONS``. ``TARGETS`` and ``SOURCES`` can each take on
various forms:

  * bare string (presumed to be a filename)
  * list of strings (presumed filenames)
  * wildcard object (``<*.c>``)
  * node object (for resources other than files)
  * list of node objects
  * variable referencing any of the above
  * concatenation of any of the above (hence ``headers + source``)

``ACTIONS`` is a newline-separated list of actions, which can be any
of:

  * string containing a shell command
  * function call (e.g. ``remove(FILE)``)
  * local variable assignment

All actions happen later, during the *build* phase. Calling
``remove()`` in a build rule doesn't remove anything while the *main*
phase is running, it just tells Fubsy to call ``remove()`` later, when
executing the actions in this build rule. However, if you call
``remove()`` outside of a build rule, it will go ahead and remove the
specified files when the *main* phase is running.

In any event, Fubsy only executes the actions in a build rule when it
determines that at least one target is out-of-date, i.e. any of the
source files have changed since the targets were last built.

You're probably wondering why that shell command uses uppercase
``$TARGET`` but lowercase ``$source``. ``$source`` is easy: it's just
a reference to the local variable ``source`` defined earlier in the
*main* phase. If we had instead called that local variable ``cfiles``,
then the command would use ``$cfiles``. ``$TARGET`` is special: it
expands to the build rule's first target file. Other special variables
that are only available in build rule actions are ``$TARGETS`` (all
targets), ``$SOURCE``, and ``$SOURCES``. We don't use ``$SOURCES`` in
this case because it includes ``*.h`` as well as ``*.c``, and you
don't pass header files to the C compiler.

So what's wrong with this example? Why is this the "WRONG WAY" to
build C programs with Fubsy? There are several problems:

  * it's not portable: ``mytool`` is the wrong filename on Windows,
    and ``cc`` is a Unix convention

  * it won't scale: for a 3-file project, it's no big deal to
    recompile the world on every change. But if ``headers`` contains
    250 header files and ``source`` 300 source files, you will feel
    the pain on every rebuild. You want an *incremental build*, where
    Fubsy rebuilds the bare minimum based on your actual source
    dependencies and which files have changed.

Incidentally, this build script isn't really *wrong*, as long as you
only care about building on Unix. It will do the job, and it
illustrates an important feature of Fubsy: you can throw together a
quick and dirty build script that gets the job done with simple core
features. The vast majority of ``Makefiles`` ever written are quick
and dirty hacks, and Fubsy aims to provide the same relaxed,
do-whatever-it-takes experience for those use cases. But when your
build script needs to grow up and get professional, Fubsy's plugin
architecture and default plugins will make life much easier than it
ever was with Make.

So what is the right way to build a C program with Fubsy?

C the right way
---------------

The right way is to use Fubsy's builtin plugin for analyzing,
compiling, and linking C libraries and programs, unsurprisingly called
``c``. Here's the complete build script::

    import c

    main {
        c.binary("myapp", <*.c>)
    }

``c.binary()`` is a *builder*: a function that defines build rules. In
this case, the rule is "build binary executable ``myapp`` from
``*.c``". There's a lot going on behind the scenes here.

  * ``"myapp"`` isn't a filename, it's the name of a binary
    executable. On Unix, it expands to filename ``myapp``, on Windows
    to ``myapp.exe``. Similar tricks apply to object files (``foo.o``
    vs. ``foo.obj``), static libraries (``libfoo.a`` vs. ``foo.lib``),
    and shared libraries (``libfoo.so`` on Linux, ``libfoo.dylib`` on
    OS X, ``foo.dll`` on Windows).

  * There are actually multiple build rules defined here: for example,
    one to compile ``myapp.c`` to ``myapp.o``, another to compile
    ``util.c`` to ``util.o``, and a third to link the two object files
    together.

  * The build rules respect header file dependencies: Fubsy (or
    rather, the ``c`` plugin) actually reads your ``*.c`` source files
    to find who includes which header files. For example, if
    ``myapp.c`` includes ``<util.h>``, then Fubsy will ensure that
    ``myapp.o`` depends on ``util.h``. You don't have to do anything;
    Fubsy just automatically takes care of C (and C++) header
    dependencies for you. Note that this is a feature of the C/C++
    plugins, and other language plugins might not be as clever. For
    example, determining compile-time dependencies for Java is
    surprisingly difficult, so the Java plugin takes a completely
    different approach to dependency analysis.

In case you're wondering, Fubsy also has excellent built-in C++
support, but the plugin is called ``cxx``. More details later.

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


A simple Java example
=====================

Since C++ and Java are two of the most widely-used programming
languages in the world, it's surprising that so few build tools even
try to support both [2]_. So before the Java programmers start to feel
left out, let's go through a similar exercise for a simple Java
project.

First, here's how the project is laid out::

  src/
    main/
      com/
        example/
          mylib/*.java
          myapp/*.java
    test/
      com/
        example/
          mylib/*.java

(This is a simplified variation on a common Java convention: test code
goes in ``src/test/``, and production, or non-test, code in
``src/main/``.)

The goal is to build all the production code into ``example.jar``,
then the test code to ``example-tests.jar``. Compiling in this order
is nice because it means you can't accidentally make production code
depend on test code: the build will fail if you do. (Eventually we
want to run the tests too, but that'll come later.)

.. [2] In fact, I'm aware of only one other build tool that supports
   both Java and C++: Gradle.

Java the naive way
------------------

First, here's the naive way to do it, using only core Fubsy features
(no plugins)::

    main {
        mainsrc = <src/main/**/*.java>
        testsrc = <src/test/**/*.java>
        mainjar = "example.jar"
        testjar = "example-test.jar"

        # recompile all production code and rebuild the production
        # jar file when any production source file changes
        mainjar: mainsrc {
            classdir = "classes/main"
            "javac -d $classdir $mainsrc"
            "jar -cf $TARGET -C $classdir ."
            remove(classdir)
        }

        # similar for the test code, but make it depend on the
        # production jar too -- i.e. recompile test code when the
        # production *bytecode* changes, not necessarily the source
        # code
        testjar: testsrc + mainjar {
            classdir = "classes/test"
            "javac -d $classdir -classpath $mainjar $testsrc"
            "jar -cf $TARGET -C $classdir ."
            remove(classdir)
        }
    }

Like the naive C example above, this works, but it could be better.
Here's how it works: first, we assign four local variables with
filenames of interest. ``mainsrc`` and ``testsrc`` are filelist
objects as above, which use wildcards to find files as late as
possible. Note the use of recursive patterns here, since Java
programmers are prone to deep package hierarchies. ``mainjar`` and
``testjar`` are of course just strings, the names of the two files
we're building. Fubsy has no idea that these are filenames; they're
just strings. It's only once a string is used as a target or source in
a build rule that it is interpreted as a filename.

Building the main ``example.jar`` is straightforward: compile a bunch
of ``.java`` files to ``.class`` files, then archive those ``.class``
files into a ``.jar`` file. We remove the intermediate directory
because it's bad form to leave around intermediate results that Fubsy
doesn't know about: those ``.class`` files are not part of Fubsy's
dependency graph, so it cannot make any use of them or even clean them
up. (Actually, it's bad form to even *create* intermediate files that
Fubsy doesn't know about; things work out better when Fubsy knows all
of your source and target files. That's tricky to do with Java,
though, so we'll hold off on doing it right until we meet the ``java``
plugin, below.)

Also, using ``remove()`` illustrates an action that is not a shell
command: you can't do this portably (``rm -rf`` on Unix, ``rmdir /s
/q`` on Windows), so instead Fubsy provides built-in support for it.

This example demonstrates local variables in build rule scope: those
two ``classdir`` variables are in fact local to each build rule, not
to the *main* phase; they exist only while the actions for each build
rule are running. (And, by the way, those actions run later, during
the *build* phase. Build rules express dependencies and specify
actions for execution during the *build* phase.)

Building ``example-test.jar`` is a bit more troublesome, and
illustrates most of the problems with this naive approach to building
Java. For starters, it largely repeats the build rule for
``example.jar``, and every programmer should know and respect the DRY
principle: Don't Repeat Yourself. Build scripts are programs too, and
should follow the same standards as your main code. But repetition is
hard to avoid when you're using core Fubsy, since the language omits
subroutines, macros, and other constructs seen in "real" programming
languages. That's deliberate: all the interesting stuff belongs in
plugins, which you can implement in a variety of "real" languages.

Another problem is that the dependency on ``example.jar`` is expressed
twice: first, we have to tell Fubsy that ``example-test.jar`` depends
on ``example.jar``, and then we have to tell ``javac`` by putting it
in the compile-time classpath. That's a Java-specific convention,
though, so of course it doesn't belong in core Fubsy. That sort of
knowledge belongs in the ``java`` plugin.


Java the right way
------------------

As with C, the right way to build your Java code is to use Fubsy's
built-in ``java`` plugin::

    import java

    main {
        mainjar = "example.jar"
        testjar = "example-test.jar"

        classdir = "classes/main"
        java.classes(classdir, <src/main/**/*.java>)
        java.jar(mainjar, classdir)

        classdir = "classes/test"
        java.classes(classdir, <src/test/**/*.java>, CLASSPATH=mainjar)
        java.jar(testjar, classdir)
    }

We're using two builders provided by the ``java`` plugin:
``classes()`` and ``jar()``. Note that builders are conventionally
named after *what* they build, not *how* they build it -- hence
``classes()`` rather than the more obvious ``javac()``. This is
largely motivated by C/C++: if ``c.binary()`` was instead named
``c.link()``, what would you call the builder that links shared
libraries? By using *what* rather than *how*, Fubsy easily
distinguishes ``c.binary()`` from ``c.sharedlibrary()``. For
consistency, that convention carries over to other plugins. It makes
sense even for Java: if you're using ``javac`` to generate annotations
rather than compile to bytecode, it's cleaner to have a separate
``annotations()`` builder than to abuse a generic ``javac()`` builder
with a clever hack that tricks it into generating annotations.

The second use of ``java.classes()`` shows our first explicit use of a
*build variable*, which is a special type of global variable defined
by plugins and used by build actions. In this case, rather than having
a single global value of ``CLASSPATH``, we override it for one
particular builder (and thus for all build rules defined by that
builder). As usual, Fubsy is relaxed about the distinction between
lists and atomic values: normally ``CLASSPATH`` is a list of filenames
and directories, but if you just pass a lone filename, that's OK.

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
