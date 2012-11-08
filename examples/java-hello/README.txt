This directory contains a simple Java project:

  src/main
    production (non-test) code for a "hello world" command-line app
  src/test
    test code for the library class in src/main

To build and test manually (on Unix):

  ./build.sh

(N.B. that shell script assumes you have JUnit 4 installed in
/usr/share/java/junit4.jar. If not, edit the script.)

To execute it:

  java -cp classes/main greet.Hello
