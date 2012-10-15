#!/bin/sh
export GOPATH=$PWD
set -ex

go tool yacc -p fu -o src/fubsy/fugrammar.go src/fubsy/fugrammar.y

# unoptimized (for debugging)
go build -v -gcflags "-N -l"
go test fubsy -v -gcflags "-N -l"
