This directory contains a simple C project. The goal is to build two
source files to an executable:

  main.c
    contains main()
    depends on greet.c
  greet.c
    utility code used by main.c
  greet.h
    declarations for functions in greet.c

To build it manually (on Unix):

  cc -o hello *.c
