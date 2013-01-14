package main

// Debug utility for Fubsy. Invoked via the main fubsy binary, but
// under a different name: it must be installed as "fubsydebug".

import (
	"fmt"
	"os"
	"path/filepath"

	"fubsy/db"
)

func debugmain() {
	prog, cmd, args, err := parseDebugArgs(os.Args)
	if err == nil {
		switch cmd {
		case "dumpdb":
			err = dumpdb(args)
		default:
			err = UsageError{"cmd [args...]", "no such command: " + cmd}
		}
	}

	switch err := err.(type) {
	case UsageError:
		fmt.Fprintln(os.Stderr, err.Usage(prog))
		os.Exit(2)
	case error:
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func parseDebugArgs(args []string) (string, string, []string, error) {
	prog := filepath.Base(os.Args[0])
	if len(args) < 2 {
		err := UsageError{"cmd [args...]", "no command supplied"}
		return prog, "", nil, err
	}
	cmd := os.Args[1]
	args = os.Args[2:]
	return prog, cmd, args, nil
}

func dumpdb(args []string) error {
	if len(args) != 1 {
		return UsageError{"dumpdb filename", "wrong number of arguments"}
	}
	bdb, err := db.OpenKyotoDB(args[0], false)
	if err != nil {
		return err
	}
	bdb.Dump(os.Stdout, "")
	return nil
}

type UsageError struct {
	usage   string
	message string
}

func (err UsageError) Error() string {
	return err.message
}

func (err UsageError) Usage(prog string) string {
	return fmt.Sprintf("usage: %s %s\n\nerror: %s",
		prog, err.usage, err.message)
}
