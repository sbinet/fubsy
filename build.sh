#!/bin/sh
export GOPATH=$PWD
set -ex

./golemon/golemon src/fubsy/fugrammar.lemon

dir=src/fubsy
for fn in fugrammar.go fugrammar_tokens.go ; do
    sed 's/package main/package fubsy/' $dir/$fn > $dir/$fn.tmp
    mv $dir/$fn.tmp $dir/$fn
done

go build -v
go test fubsy -v
