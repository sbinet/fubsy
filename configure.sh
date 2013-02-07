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
    tagif kyotodb "pkg-config --cflags kyotocabinet"

    # Probe for "python" last because on Arch Linux, "python" is
    # Python 3. We want Python 2.6 or 2.7 (I suspect -- have not tried
    # older versions).
    tagif python \
        "pkg-config --cflags python2" \
        "pkg-config --cflags python-2.6" \
        "pkg-config --cflags python-2.7" \
        "pkg-config --cflags python"

    echo ""
    echo "build tags:"
    (cd $tagdir && echo *)
}

getdeps() {
    echo ""
    echo "downloading dependencies ..."

    # directory for the "stage 1" build, done with shell scripts
    # rather then with Fubsy itself
    build1=".build/1"

    export GOPATH="$top/$build1"
    if tagset python; then
        run "go get -v -d github.com/sbinet/go-python/pkg/python"
    fi

    # we build separate from download mainly because of go-python,
    # which has a Makefile for good reasons of its own
    # (also, perhaps building should be done in build.sh...?)

    echo ""
    echo "building dependencies ..."
    mkdir -p $build1/bin
    mkdir -p $build1/pkg/$goplatform/github.com

    rm -rf pkg/$goplatform/github.com/cznic
    run "GOPATH=$top go install -v github.com/cznic/..."
    run "mv pkg/$goplatform/github.com/cznic $build1/pkg/$goplatform/github.com/."
    run "mv bin/golex $build1/bin/."

    rm -rf pkg/$goplatform/github.com/axw
    run "GOPATH=$top go install -v github.com/axw/gocov/gocov"
    run "mv pkg/$goplatform/github.com/axw $build1/pkg/$goplatform/github.com/."
    run "mv bin/gocov $build1/bin/."

    if tagset python; then
        run "make -C $build1/src/github.com/sbinet/go-python/pkg/python install"
    fi
}

# utility functions

run() {
    echo $1
    echo '$' $1 >> $log
    eval $1 >> $log 2>&1
    status=$?
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

tagif() {
    tag=$1 ; shift
    for cmd in "$@"; do
        if check "$cmd"; then
            touch $tagdir/$tag
            return
        fi
    done
}

tagset() {
    test -f $tagdir/$1
}

# main program

setup
probe
getdeps
