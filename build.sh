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

$golex -o src/fubsy/dsl/fulex.go src/fubsy/dsl/fulex.l
gofmt -w src/fubsy/dsl/fulex.go

go tool yacc -p fu -o src/fubsy/dsl/fugrammar.go src/fubsy/dsl/fugrammar.y

# uncomment this to run benchmarks
#benchopt="-test.bench=.*"

# unoptimized (for debugging)
packages="dsl types dag runtime"
#packages="dsl"
#packages="runtime"
for pkg in $packages; do
    go install -v -gcflags "-N -l" fubsy/$pkg
    go test -v -gcflags "-N -l" -i fubsy/$pkg
    go test -v -gcflags "-N -l" -c fubsy/$pkg
    ./$pkg.test -test.v=true $benchopt $tests
done

go build -v -gcflags "-N -l"
