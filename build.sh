#!/bin/sh
export GOPATH=$PWD
set -ex
go build
go test fubsy -v
