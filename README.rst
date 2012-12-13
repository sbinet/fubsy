Fubsy: The Universal Build Tool
===============================

Fubsy is a tool for efficiently building software. If you need to
minimally rebuild a bunch of target files from a bunch of source files
by following a bunch of rules, Fubsy is what you need. If you prefer
formal abstract language, Fubsy is an engine for conditional execution
of actions based on the dependencies between related resources.

Building
--------

Fubsy is written in Go, and uses the gc compiler to build. If you're
on Unix and you already have Go installed, just run ::

    ./build.sh

Otherwise, see the online docs: http://fubsy.gerg.ca/develop/.

Using
-----

See the user guide in ``doc/guide`` (online at http://fubsy.readthedocs.org/).

Currently, Fubsy is in the very early stages of development. You can
write tiny toy build scripts with it, but there is a lot of work to do
before it's ready for the real world.

Contributing
------------

The main purpose of this release is to attract developers who want to
help shape Fubsy into a world-class build tool. Please see
http://fubsy.gerg.ca/develop/ for more information.

Author, copyright, license
--------------------------

Fubsy was written by Greg Ward <greg at gerg dot ca>.

Copyright © 2012, Greg Ward. All rights reserved.

Use of this software is governed by a BSD-style license that can be
found in the LICENSE.txt file.
