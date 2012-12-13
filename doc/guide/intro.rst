Introduction
============

Fubsy is a tool for efficiently building software. Roughly speaking,
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
compiling ``mytool.c`` and ``util.c``, and then linking the two object
files together. More importantly, you want to perform the minimum
necessary work whenever source files change: if you modify
``mytool.c``, then recompile ``mytool.o`` and relink the executable.
But if you modify ``util.h``, you may need to recompile both
``util.o`` and ``main.o`` before relinking. This is exactly the sort
of problem that Fubsy is designed for.

(At this point, outraged Windows programmers might point out that they
build ``mytool.obj`` and ``mytool.exe`` This platform variation is a
quirk of C/C++ that Fubsy's C/C++ plugins handle, but which core Fubsy
knows nothing about.)

Disclaimer
----------

.. note:: Currently, this document is more of a specification than a
          description of actual software. Many of the features
          described here have barely even been thought through, never
          mind implemented and tested. Mentally add a "not implemented
          yet" footnote to every sentence in this guide, and you won't
          be too far off from the truth. I've tried to help by adding
          explicit "this actually works" and "not implemented yet"
          notes here and there, but don't be surprised if Fubsy
          doesn't behave quite as the document promises.

          Furthermore, it's quite likely that the final product will
          differ considerably from the description in this document;
          that's just the nature of software. Consider this an
          invitation to `join in the development of Fubsy
          <http://fubsy.gerg.ca/develop/>`_ and influence how it will
          turn out.

Similar tools
-------------

Of course, Fubsy is hardly the first piece of software that attempts
to tackle this problem. Every C programmer is familiar with Make,
which does a reasonable job for small-to-medium C/C++ projects on
Unix-like systems (if you ignore the difficulty with header
dependencies). However, Make has awkward syntax, poor extensibility,
and confusing semantics, which have led many people over the years to
paper over its difficulties by writing Makefile generators and the
like.

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
