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

# unoptimized (for debugging)
packages="dsl dag runtime"
for pkg in $packages; do
    go install -v -gcflags "-N -l" fubsy/$pkg
    go test -v -gcflags "-N -l" -i fubsy/$pkg
    go test -v -gcflags "-N -l" -c fubsy/$pkg
    ./$pkg.test -test.v=true -test.bench='.*' $tests
done

go build -v -gcflags "-N -l"
