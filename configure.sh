#!/bin/sh

# auto-configuration script for Fubsy; needed until Fubsy has builtin
# auto-configuration capabilities
#
# the goals here are
# 1) probe the build system to see what features
#    are present, recording the results in files under .build
# 2) download dependencies (mainly Go libraries, not packages we expect
#    to find installed)
#
# To clarify, Fubsy has three kinds of dependencies (apart from Go
# itself):
#   * optional, written in C or C++ (Kyoto Cabinet, Python)
#   * optional, written in Go (typically wrappers for optional
#     C/C++ dependencies)
#   * required, written in Go (golex, go-bit, ...).
#
# We generally expect optional C/C++ dependencies to be installed
# separately on the build system, and carry on without them if they
# are not there. Go dependencies are either included directly in
# Fubsy's source repository or downloaded here: either way, if they
# are required, the build process guarantees they will be present (or
# the build will fail).

top=`pwd`
log=".build/config/log"
tagdir=".build/tags"
goos=`go env GOOS`
goarch=`go env GOARCH`
goplatform="${goos}_${goarch}"

# high-level functions

setup() {
    rm -rf .build
    mkdir -p $tagdir .build/config
}

probe() {
    echo "probing build system ..."
    pkgtag kyotodb kyotocabinet

    # Probe for "python" last because on Arch Linux, "python" is
    # Python 3. We want Python 2.6 or 2.7 (I suspect -- have not tried
    # older versions).
    pkgtag python python2 python-2.6 python-2.7 python

    echo ""
    echo "build tags:"
    (cd $tagdir && echo *)

    cgoflags $tagdir/*
}

getdeps() {
    echo ""
    echo "downloading dependencies ..."

    # directory for the "stage 1" build, done with shell scripts
    # rather than with Fubsy itself
    build1=".build/1"
    mkdir -p $build1

    export GOPATH="$top/$build1"
    if tagset python; then
        run "go get -v -d github.com/sbinet/go-python"
    fi

    run "go get -v -d github.com/axw/gocov"
    run "go get -v -d github.com/cznic/golex"

    # we build separate from download mainly because of go-python,
    # which has a Makefile for good reasons of its own
    # (also, perhaps building should be done in build.sh...?)

    echo ""
    echo "building dependencies ..."
    mkdir -p $build1/bin
    mkdir -p $build1/pkg/$goplatform/github.com

    rm -rf pkg/$goplatform/github.com/cznic
    run "go install -v github.com/cznic/golex"

    rm -rf pkg/$goplatform/github.com/axw
    run "go install -v github.com/axw/gocov/gocov"

    if tagset python; then
        run "make -C $build1/src/github.com/sbinet/go-python install"
    fi
}

# utility functions

run() {
    cmd="$1"
    echo $cmd
    echo '$' $cmd >> $log
    eval $cmd >> $log 2>&1
    status=$?
    if [ $status -ne 0 ]; then
        echo "error: command failed: see $log for details" >&2
        exit $status
    fi
    echo >> $log
}

# run, redirecting command's output to a dedicated file (as well as $log)
capture() {
    cmd="$1"
    rfile="$2"
    echo $cmd
    echo '$' $cmd >> $log
    eval $cmd > $rfile 2>&1
    status=$?
    cat $rfile >> $log
    if [ $status -ne 0 ]; then
        echo "error: command failed: see $log for details" >&2
        exit $status
    fi
    echo >> $log
}

check() {
    echo -n "$1 ... "
    echo '$' $1 >> $log
    eval $1 >> $log 2>&1
    status=$?
    echo >> $log

    if [ $status -eq 0 ]; then
        echo "ok"
    else
        echo "fail"
    fi
    return $status
}

pkgtag() {
    tag="$1" ; shift
    for pkg in "$@"; do
        cmd="pkg-config $pkg"
        if check "$cmd"; then
            echo $pkg > $tagdir/$tag
            return
        fi
    done
}

cgoflags() {
    tagfiles="$@"
    pkgnames=`cat $tagfiles`
    capture "pkg-config --cflags $pkgnames" .build/config/cgo-cflags
    capture "pkg-config --libs $pkgnames" .build/config/cgo-ldflags
}

tagset() {
    test -f $tagdir/$1
}

# main program

setup
probe
getdeps
