Fubsy, the scripting language
=============================

By now you've probably noticed that Fubsy is a simple scripting
language as well as a build tool. It has constructs that are familiar
from other programming languages, like variables, strings, lists, and
function calls. It's missing lots of functionality that you would
expect in a general-purpose language, like numbers, arithmetic, loops,
and user-defined functions. And it has some features that are vaguely
like subroutines, but geared towards the particular needs of a build
tool like Fubsy: phases and build rules.

The formal specification for Fubsy's scripting language doesn't exist
yet, so here's an informal guide.

Top-level elements
------------------

At the top level, a Fubsy script contains three elements: import
statements, inline plugins, and phases::

    import PLUGIN

    plugin LANG {{{
        CONTENT
    }}}

    PHASE {
        STATEMENT
        ...
    }

There can be any number of each element, and they can be intermixed in
any order::

    import java

    plugin python {{{
        def pyhello():
            # println() is a Fubsy function exposed to plugins
            println("hello from a python plugin")
    }}}

    import c

    clean {
        jshello()
        remove("junkfile")
    }

    plugin javascript {{{
        func jshello() {
            println("hello from a javascript plugin")
        }
    }}}

    main {
        pyhello()
        c.binary("bin/myapp", <src/*.c>)
        java.classes("classes/foo", <src/foo/**/*.java>)
        touch("junkfile")
    }

However, good taste suggests that imports should come first, followed
by inline plugins, followed by phases in a logical order. Here's an
equivalent build script::

    import java
    import c

    plugin python {{{
        def pyhello():
            # println() is a Fubsy function exposed to plugins
            println("hello from a python plugin")
    }}}

    plugin javascript {{{
        func jshello() {
            println("hello from a javascript plugin");
        }
    }}}

    main {
        pyhello()
        c.binary("bin/myapp", <src/*.c>)
        java.classes("classes/foo", <src/foo/**/*.java>)
        touch("junkfile")
    }

    clean {
        jshello()
        remove("junkfile")
    }

.. note:: Don't expect this to work with Fubsy 0.0.1. The parser
          supports all of the syntax shown here, but almost none of
          the required backend code has been implemented yet.

.. note:: There's no provision for builds with multiple build scripts
          (aka "hierarchical builds") yet. I'm leaning towards a
          simple ``include FILENAME`` syntax (like Make), with
          automatic reinterpretation of filenames in child scripts
          (like SCons). Join the `fubsydev mailing list
          <http://fubsy.gerg.ca/lists/>`_ if you want to help shape
          the design of Fubsy!

(Some) whitespace is significant
--------------------------------

End-of-line is syntactically significant: it's the delimiter between
statements. The only difference between ::

    # invalid syntax
    import a import b

and ::

    # valid syntax
    import a
    import b

is the newline after ``import a``. The same applies to statements
inside phases and build rules.

Less obviously, the relationship between curly braces and newlines is
fixed by the grammar::

    # valid syntax
    main {
        a = "a"
    }

    # invalid syntax
    main
    {
        a = "a"
    }

    # invalid syntax
    main {
        a = "a" }

    # valid syntax (special case for empty phases)
    main { }

Build rules have similar syntax.

Newline is the *only* type of whitespace that is significant, though:
Fubsy does not care how you indent your code. You can use tabs or
spaces or both or (shudder) no indentation at all. (But *I* care:
please indent with four spaces, so all Fubsy scripts will be
consistent.)

.. note:: I like the idea of consistent style being enforced by the
          grammar, but clearly I didn't have the guts to go as far as
          enforcing indentation. If you strongly disagree with this
          design choice, one way or the other, please join the
          `fubsydev mailing list <http://fubsy.gerg.ca/lists/>`_ and
          discuss!

Imports
-------

Fubsy will support *external* and *inline* plugins. One import
statement loads one external plugin using a dot-delimited name::

    import c
    import java.eclipse
    import foo.bar.baz.wing.ding

The effect of each ``import`` is to add one name to the local
namespace of the current script: in this case, ``c``, ``eclipse``, and
``ding``. Each plugin provides values that you can use in your build
script::

    main {
        c.binary("app", "main.c")
        println(ding.TOOLNAME)
    }

Precisely what a plugin provides is entirely up to the plugin.

.. note:: Apart from syntactic support, this is completely
          unimplemented in Fubsy 0.0.1.

Inline plugins
--------------

Fubsy deliberately does not provide general programming features such
as numbers, arithmetic, loops, or user-defined functions. That's what
inline plugins are for. There are plenty of good high-level
general-purpose languages out there already, so it seems silly to
design and implement yet another general-purpose language for a
special-purpose build tool. (And Fubsy deliberately does not use an
existing general-purpose language for its syntax, because that would
limit it to fans of that particular language. Fubsy aims to be a
*universal* build tool, and inline plugins are a key part of achieving
that goal. If you want SCons/Rake/Waf/Gradle, you know where to find
them.)

The syntax for an inline plugin is ::

    plugin LANGUAGE {{{CONTENT}}}

where ``LANGUAGE`` is a short identifier like "python" or "javascript"
and ``CONTENT`` is any sequence of bytes, except for ``}}}``.

The language tells Fubsy how to interpret the content. If you put
JavaScript code in a plugin marked ``python``, then Fubsy will happily
fire up a Python interpreter, ask Python to parse your code, and fail.

Whitespace inside the triple-brace delimiters is ignored and passed
verbatim to the plugin interpreter, *except* that common leading
whitespace is trimmed. That is, if every line of ``CONTENT`` starts
with (at least) four spaces, then four spaces will be trimmed from
``CONTENT`` before attempting to parse it. That lets you indent your
inline plugin content without angering indentation-sensitive languages
like Python.

Functions and values defined by inline plugins will be available to
the build script directly. See the example above, under "Top-level
elements".

.. note:: Apart from syntactic support, this is completely
          unimplemented in Fubsy 0.0.1.

Phases
------

A phase is just a sequence of statements::

   NAME {
       STATEMENT
       ...
   }

where ``NAME`` is an identifier like ``main``, ``clean``, ``options``,
etc.

A statement can be one of the following:

  * a variable assignment, like ::

        src = <src/main/**/*.java>
        java.JAVAC = "/usr/bin/javac"
        java.CLASSPATH = ["lib/util.jar", "lib/stuff.jar"]

  * an expression, like ::

        src.exclude("**/Stub*.java")
        pyhello()
        mkdir(builddir + "/" + "bin")

  * a build rule, like ::

        "app.jar": <classes/app/**/*.class> {
            "jar -cf ../../$TARGET -C classes/app .
        }

Every Fubsy build script must contain a *main* phase, which defines
sources and targets and the relationships between them. See `phases
<phases.html>`_ for more information on the phases that Fubsy will
eventually implement and the relationships between them.

Local and global variables
--------------------------

By default, variables are local to the current script, and available
to all phases in it::

    main {
        junkfile = "tmp/junk.dat"
        touch(junkfile)
    }

    clean {
        remove(junkfile)
    }

Thus, while phases look like a scoping mechanism, they aren't. They're
really a mechanism for specifying what happens at different times in
the process of a build. That's why they are *sort of* like
subroutines, but not really. (They also don't have parameters or
return values, and you don't have much control over when they run.)

Variables can also be defined in a build rule, in which case they are
local to that build rule only::

    main {
        "outfile": "infile" {
            tmpfile = "$TARGET.tmp"
            "./process $SOURCE > $tmpfile"
            rename(tmpfile, TARGET)
        }

        # runtime error: 'tmpfile' not defined
        println(tmpfile)
    }

Thus, build rules *are* a scoping mechanism. But they are primarily a
means for you to write code that isn't run until the *build* phase,
and only runs if any of the rule's targets are stale or missing.

A future version of Fubsy will support hierarchical builds where a
top-level build script includes child scripts for building code in
subdirectories. When that happens, Fubsy will also grow support for
global variables that are visible to all scripts in the same process.
Until that point, there's not much point in implementing global
variables.

Value expansion
---------------

All values in Fubsy -- strings, lists, and filefinders -- are subject
to *expansion*. The precise meaning of expansion varies according to
the data type, but in general it means converting a value from the
form initially seen in the build script to the form that will be
needed in order to actually build targets.

For example, the filefinder value ``<*.c>`` might expand to a list
like ::

    ["main.c", "util.c", "stuff.c"]

and the string ::

    "$CC -o $TARGET $sources"

might expand to ::

    "/usr/bin/cc -o app main.c util.c stuff.c"

Expanding a list just means expanding its member values recursively,
and flattening the result. For example, the list ::

    [<*.c>, "hello $audience", <include/*.h>]

consists of three values which might respectively expand to ::

    ["main.c", "util.c", "stuff.c"]
    "hello world"
    ["include/util.h", "include/stuff.h"]

But list expansion results in a flattened value ::

    ["main.c",
     "util.c",
     "stuff.c",
     "hello world",
     "include/util.h",
     "include/stuff.h"]

Fubsy is perfectly capable of representing deeply nested data
structures, but it generally flattens lists whenever it can. Fubsy is
not a general-purpose programming language, and flat lists tend to be
more convenient in build scripts.

In the absence of explicit expansion, by you or by plugin code that
you call, values are expanded in the *build* phase. Values that are
nodes in the dependency graph (a common use of filefinders) are
expanded early in the build phase, when Fubsy converts the initial
dependency graph to its final form. Other values (e.g. command
strings) are not expanded until right before the command is executed.
Consider this build script::

    main {
        flags = "-O2"
        "myapp": <*.c> {
            flags = "-O0 -Wall"
            "cc $flags $SOURCES -o $TARGET"
        }
    }

Expanding command strings at the last possible moment means ``$flags``
expands to ``-O0 -Wall``, as you would expect. It's also essential for
automatic variables like ``SOURCES`` and ``TARGET`` to work.

Summary
-------

Fubsy's scripting language provides the following familiar features,
which should be familiar from most general-purpose programming
languages:

  * variables
  * data types: strings, lists
  * expressions, including function calls

The scoping rules for variables are a bit odd:

  * most variables are local to current script
  * but phases are not scopes: a variable defined in *main* is
    visible in *build*, *clean*, etc.
  * each build rule is a scope and has local variables

Fubsy also has some distinctive features:

  * filefinder objects for wildcards (``<src/*.c>``)
  * expansion of variables embedded in strings (``"$CC -o $TARGET"``)

These may look familiar from Unix shell programming, but there's a key
difference: in Fubsy, wildcards and strings are expanded as late as
possible.

Finally, Fubsy deliberately omits a number of features found in any
general-purpose programming language:

  * numbers
  * arithmetic
  * loops
  * user-defined functions
  * logic ("a or b and not c")
  * conditionals (if/then/else)

Fubsy is not a general-purpose language. If you need those things,
you'll have to write an inline plugin in an existing language (when
Fubsy grows support for inline plugins!).

(Actually, I suspect Fubsy will have to provide conditionals and logic
eventually. The point of the *options* and *configure* phases will be
to make the build vary according to user wishes and the state of the
build system. User-defined options won't be very useful if they don't
provide a way for you to enable/disable parts of your build, and
explicit conditional constructs are the obvious answer there.)
