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

tagdir=".build/tags"
buildtags=`cd $tagdir && echo *`
tagflag="-tags='$buildtags'"

exclude="fubsy/testutils,\
github.com/stretchrcom/testify/assert,\
code.google.com/p/go-bit/bit,\
github.com/ogier/pflag,\
github.com/sbinet/go-python"

echo "testing packages: $packages"
build1=".build/1"
set -e
for pkg in $packages; do
    json=.build/coverage-`basename $pkg`.json
    report=.build/coverage-`basename $pkg`.txt
    run "$build1/bin/gocov test $tagflag -exclude $exclude $pkg > $json"
    run "$build1/bin/gocov report $json > $report"
done
