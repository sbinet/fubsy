#!/bin/sh

showvar() {
    name=$1
    val=`eval echo \\$$name`
    echo "$name=\"$val\" ; export $name"
}

run() {
    echo $1
    eval $1
}

checkexists() {
    test=$1   # "-d", "-f", etc.
    file=$2
    if [ ! $test $file ]; then
        echo "error: $file not found (did you run ./configure.sh?)" >&2
        exit 1
    fi
}

# stage 1 build dir, created by configure.sh
build1=".build/1"
checkexists -d $build1

tests=""
if [ $# -eq 1 ]; then
    tests="-test.run=$1"
fi

top=`pwd`
export GOPATH=$top:$top/.build/1
showvar GOPATH

# set build tags based on what configure.sh found when it probed
tagdir=".build/tags"
checkexists -d $tagdir
buildtags=`cd $tagdir && echo *`

# likewise, set CGO_* based on configure.sh's output
configdir=".build/config"
checkexists -f $configdir/cgo-cflags
CGO_CFLAGS=`cat $configdir/cgo-cflags | sed 's/ *$//' | tr -d '\n'`
export CGO_CFLAGS
showvar CGO_CFLAGS

checkexists -f $configdir/cgo-ldflags
CGO_LDFLAGS=`cat $configdir/cgo-ldflags | sed 's/ *$//' | tr -d '\n'`
export CGO_LDFLAGS
showvar CGO_LDFLAGS

set -e

golex="$build1/bin/golex"
checkexists -f $golex

gocov="$build1/bin/gocov"
checkexists -f $gocov

run "$golex -o src/fubsy/dsl/fulex.go src/fubsy/dsl/fulex.l"
run "go tool yacc -p fu -o src/fubsy/dsl/fugrammar.go src/fubsy/dsl/fugrammar.y"
run "gofmt -w src/fubsy/dsl/fulex.go src/fubsy/dsl/fugrammar.go"

run "python genplugins.py src/fubsy/runtime/builtins.go src/fubsy/plugins/empython.c"

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
