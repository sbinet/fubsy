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

kyotodb=src/fubsy/db/kyotodb
if [ ! -f "$kyotodb.go" ]; then
    set +e
    run "pkg-config --silence-errors --cflags kyotocabinet"
    status=$?
    set -e
    if [ $status -eq 0 ]; then
        run "ln -sf kyotodb.go.real ${kyotodb}.go"
        run "ln -sf kyotodb_test.go.real ${kyotodb}_test.go"
    else
        run "ln -sf kyotodb.go.fake ${kyotodb}.go"
    fi
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

# uncomment this to get test coverage
#coverage=y

# only explicitly build packages with tests
packages="fubsy/log fubsy/dsl fubsy/types fubsy/dag fubsy/db fubsy/build fubsy/runtime fubsy"
#packages="fubsy/dsl"
#packages="fubsy/types"
#packages="fubsy/dag"
#packages="fubsy/build"
#packages="fubsy/runtime"

run "go install -v -gcflags '-N -l' $packages"
run "ln -sf fubsy bin/fubsydebug"
run "go test -v -gcflags '-N -l' -i $packages"

if [ "$coverage" ]; then
    for pkg in $packages; do
        json=coverage-`basename $pkg`.json
        report=coverage-`basename $pkg`.txt
        run "./bin/gocov test \
            -exclude fubsy/testutils,github.com/stretchrcom/testify/assert,code.google.com/p/go-bit/bit \
            $pkg > $json"
        run "./bin/gocov report $json > $report"
    done
else
    run "go test -gcflags '-N -l' $benchopt $packages $tests"
fi

run "go vet $packages"

# make sure all source files are gofmt-compliant
echo "gofmt -l src/fubsy"
needfmt=`gofmt -l src/fubsy`
if [ "$needfmt" ]; then
    echo "error: gofmt found non-compliant files"
    echo "you probably need to run: gofmt -w" $needfmt
    exit 1
fi
