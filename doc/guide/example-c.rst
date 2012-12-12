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
a reference to the variable ``source`` defined earlier in the *main*
phase. If we had instead called that variable ``files``, then
the command would use ``$files``. ``$TARGET`` is special: it expands
to the build rule's first target file. Other special variables that
are only available in build rule actions are ``$TARGETS`` (all
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
