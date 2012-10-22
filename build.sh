#!/bin/sh
export GOPATH=$PWD
set -ex

golex=bin/golex
if [ ! -f $golex ]; then
    go install github.com/cznic/golex
fi

$golex -o src/fubsy/fulex.go src/fubsy/fulex.l
gofmt -w src/fubsy/fulex.go

go tool yacc -p fu -o src/fubsy/fugrammar.go src/fubsy/fugrammar.y

# unoptimized (for debugging)
go build -v -gcflags "-N -l"
go test -c fubsy -v -gcflags "-N -l"
./fubsy.test -test.v=true
