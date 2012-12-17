A simple Java example
=====================

Since C and Java are two of the most widely-used programming
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
   Java and C/C++: Gradle.

Java the naive way
------------------

.. note:: Not working yet. On the to-do list for Fubsy 0.0.1.

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
            mkdir(classdir)
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
            mkdir(classdir)
            "javac -d $classdir -classpath $mainjar $testsrc"
            "jar -cf $TARGET -C $classdir ."
            remove(classdir)
        }
    }

Like the naive C example above, this works, but it could be better.
Here's how it works: first, we assign four variables with filenames of
interest. ``mainsrc`` and ``testsrc`` are filefinder objects as above,
which use wildcards to find files as late as possible. Note the use of
recursive patterns here, since Java programmers are prone to deep
package hierarchies. ``mainjar`` and ``testjar`` are just string
variables, the names of the two files we're building. Fubsy has no
idea that these are filenames; they're just strings. It's only once a
string is used as a target or source in a build rule that it is
interpreted as a filename.

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

This example demonstrates variables local to a build rule: those two
``classdir`` variables are in fact distinct and not visible outside of
the build rules. They exist only while the actions for each build rule
are running. (And, by the way, those actions run later, during the
*build* phase. That's the whole point of build rules, after all: to
specify actions that might run in the build phase, if at least one
source has changed.)

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
on ``example.jar``, and then we have to tell ``javac`` the same thing
by putting it in the compile-time classpath. That's a Java-specific
convention, though, so of course it doesn't belong in core Fubsy. That
sort of knowledge belongs in the ``java`` plugin.


Java the right way
------------------

.. note:: Not implemented yet. First we need to figure out the
          architecture for plugins, then start implementing useful
          plugins.

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
*build variable*, which is a special type of variable defined by
plugins and used by build actions. In this case, rather than having a
single value of ``CLASSPATH``, we override it for one particular

builder (and thus for all build rules defined by that builder). As
usual, Fubsy is relaxed about the distinction between lists and atomic
values: normally ``CLASSPATH`` is a list of filenames and directories,
but if you just pass a lone filename, that's OK.
