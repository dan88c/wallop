package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dan88c/wallop/core-operator/pkg/graph"
	"github.com/dan88c/wallop/core-operator/pkg/guard"
	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

const (
	Version       = "1.1.000"
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
