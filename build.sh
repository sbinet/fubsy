#!/bin/sh

run() {
    echo $1
    eval $1
}

tests=""
if [ $# -eq 1 ]; then
    tests="-test.run=$1"
fi

export GOPATH=$PWD
set -e

# by default, build with no optional features -- they will be enabled
# based on what exists on the build system
buildtags=""

set +e
run "pkg-config --silence-errors --cflags kyotocabinet"
status=$?
set -e
if [ $status -eq 0 ]; then
    buildtags="$buildtags kyotodb"
fi

golex=bin/golex
if [ ! -f $golex ]; then
    run "go install github.com/cznic/golex"
fi

gocov=bin/gocov
if [ ! -f $gocov ]; then
    run "go install github.com/axw/gocov/gocov"
fi

run "$golex -o src/fubsy/dsl/fulex.go src/fubsy/dsl/fulex.l"
run "go tool yacc -p fu -o src/fubsy/dsl/fugrammar.go src/fubsy/dsl/fugrammar.y"
run "gofmt -w src/fubsy/dsl/fulex.go src/fubsy/dsl/fugrammar.go"

# uncomment this to run benchmarks
#benchopt="-test.bench=.*"

packages="fubsy/..."
#packages="fubsy/dsl"
#packages="fubsy/types"
#packages="fubsy/dag"
#packages="fubsy/build"
#packages="fubsy/runtime"

tagflag="-tags='$buildtags'"
run "go install -v -gcflags '-N -l' $tagflag $packages"
run "ln -sf fubsy bin/fubsydebug"
run "go test -v -i -gcflags '-N -l' $tagflag $packages"
run "go test -gcflags '-N -l' $tagflag $benchopt $packages $tests"

run "go vet $packages"

# make sure all source files are gofmt-compliant
echo "gofmt -l src/fubsy"
needfmt=`gofmt -l src/fubsy`
if [ "$needfmt" ]; then
    echo "error: gofmt found non-compliant files"
    echo "you probably need to run: gofmt -w" $needfmt
    exit 1
fi
