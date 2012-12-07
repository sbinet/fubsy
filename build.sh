#!/bin/sh

tests=""
if [ $# -eq 1 ]; then
    tests="-test.run=$1"
fi

export GOPATH=$PWD
set -ex

golex=bin/golex
if [ ! -f $golex ]; then
    go install github.com/cznic/golex
fi

gocov=bin/gocov
if [ ! -f $gocov ]; then
    go install github.com/axw/gocov/gocov
fi

$golex -o src/fubsy/dsl/fulex.go src/fubsy/dsl/fulex.l
gofmt -w src/fubsy/dsl/fulex.go

go tool yacc -p fu -o src/fubsy/dsl/fugrammar.go src/fubsy/dsl/fugrammar.y

# uncomment this to run benchmarks
#benchopt="-test.bench=.*"

# uncomment this to get test coverage
#coverage=y

# only explicitly build packages with tests
packages="fubsy/dsl fubsy/types fubsy/dag fubsy/runtime"
#packages="fubsy/dsl"
#packages="fubsy/dag"
#packages="fubsy/runtime"

go install -v -gcflags "-N -l" $packages
go test -v -gcflags "-N -l" -i $packages

if [ "$coverage" ]; then
    for pkg in $packages; do
        json=coverage-`basename $pkg`.json
        report=coverage-`basename $pkg`.txt
        ./bin/gocov test \
            -exclude fubsy/testutils,github.com/stretchrcom/testify/assert,code.google.com/p/go-bit/bit \
            $pkg > $json
        ./bin/gocov report $json > $report
    done
else
    go test -v -gcflags "-N -l" $benchopt $packages $tests
fi

go install -v -gcflags "-N -l" fubsy
