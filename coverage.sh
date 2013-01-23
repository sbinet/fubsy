#!/bin/sh

# run Fubsy unit tests with coverage analysis
# assumes all packages are built and all tests pass (i.e.
# build.sh succeeds)
#
# usage:
#   ./coverage.sh [package] ...
# (if no packages are given, test all fubsy packages)

run() {
    echo $1
    eval $1
}

if [ $# -eq 0 ]; then
    packages=`find src/fubsy -name '*_test.go' \
              | sed 's/^src\///; s/\/[a-z_\.]*\.go//' \
              | sort -u \
              | tr '\n' ' '`
else
    packages=$*
fi

exclude="fubsy/testutils,\
github.com/stretchrcom/testify/assert,\
code.google.com/p/go-bit/bit,\
github.com/ogier/pflag"

echo "testing packages: $packages"
set -e
for pkg in $packages; do
    json=coverage-`basename $pkg`.json
    report=coverage-`basename $pkg`.txt
    run "./bin/gocov test -exclude $exclude $pkg > $json"
    run "./bin/gocov report $json > $report"
done
