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

func defaultRegistry() string {
	if p := os.Getenv("WALLOP_ROOT"); p != "" {
		return filepath.Join(p, "config", "tool_registry.yaml")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "config/tool_registry.yaml"
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		cand := filepath.Join(dir, "config", "tool_registry.yaml")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	return "config/tool_registry.yaml"
}

func pythonBin() string {
	for _, name := range []string{"python3", "python", "py"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

func fail(code int, msg string) {
	fmt.Fprintf(os.Stderr, "ERROR code=%d %s\n", code, msg)
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(map[string]any{"ok": false, "code": code, "error": msg})
}

func cmdTOC(args []string) int {
	fs := flag.NewFlagSet("toc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	full := fs.Bool("full", false, "")
	tag := fs.String("tag", "", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	out, err := registry.RenderTOC(reg, registry.TOCOptions{Full: *full, Tag: *tag})
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	fmt.Print(out)
	return exitOK
}
