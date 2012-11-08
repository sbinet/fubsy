#!/bin/sh -ex
rm -rf classes
mkdir -p classes/main classes/test
javac -d classes/main `find src/main -name "*.java"`
javac -d classes/test -cp classes/main:/usr/share/java/junit4.jar `find src/test -name "*.java"`

java -cp classes/main:classes/test:/usr/share/java/junit4.jar org.junit.runner.JUnitCore greet.GreeterTest
