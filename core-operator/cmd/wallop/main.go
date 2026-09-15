package main

import (
	"fmt"
	"os"
)

const (
	Version        = "1.2.000"
	exitOK         = 0
	exitInvalid    = 1
	exitUsage      = 2
	exitUnknown    = 3
	exitBadPayload = 4
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitUsage)
	}

	switch os.Args[1] {
	case "toc":
		os.Exit(cmdTOC(os.Args[2:]))
	case "path", "which":
		os.Exit(cmdPath(os.Args[2:]))
	case "desc", "info", "describe":
		os.Exit(cmdDesc(os.Args[2:]))
	case "run":
		os.Exit(cmdRun(os.Args[2:]))
	case "log":
		os.Exit(cmdLog(os.Args[2:]))
	case "guard", "validate":
		os.Exit(cmdGuard(os.Args[2:]))
	case "graph":
		os.Exit(cmdGraph(os.Args[2:]))
	case "register":
		os.Exit(cmdRegister(os.Args[2:]))
	case "doctor":
		os.Exit(cmdDoctor(os.Args[2:]))
	case "version", "-v", "--version":
		fmt.Printf("wallop %s\n", Version)
		os.Exit(exitOK)
	case "help", "-h", "--help":
		usage()
		os.Exit(exitOK)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		usage()
		os.Exit(exitUsage)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `wallop %s — typed operator shell

Commands:
  wallop toc [--full] [--tag name]
  wallop path --tool <name>
  wallop desc --tool <name>
  wallop run --tool <name> --payload <json>
  wallop log [--last] [--tool name]
  wallop guard --tool <name> [--payload <json>]
  wallop graph --vault <path> [--query <keyword>]
  wallop register --entry <script> --name <id> [--keep-path] [--venv path] [--runtime python|shell|exec]
  wallop doctor [--registry path]
  wallop version

Exit codes:
  0 ok
  1 invalid call / doctor found errors
  2 usage
  3 unknown tool / registry
  4 unreadable payload
  5 child process failed (see logs/)
`, Version)
}
