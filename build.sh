#!/bin/sh
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
go install -v -gcflags "-N -l" fubsy
go test -v -gcflags "-N -l" -i fubsy/dsl
go test -v -gcflags "-N -l" -c fubsy/dsl
./dsl.test -test.v=true

go build -v -gcflags "-N -l"
