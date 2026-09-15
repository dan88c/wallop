package main

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dan88c/wallop/core-operator/pkg/graph"
	"github.com/dan88c/wallop/core-operator/pkg/guard"
	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

func cmdDoctor(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	if dups := reg.DuplicateNames(); len(dups) > 0 {
		fail(exitInvalid, "duplicate tool names: "+strings.Join(dups, ", "))
		return exitInvalid
	}
	root := registry.RepoRootFromRegistry(*regPath)
	script := filepath.Join(root, "scripts", "doctor.py")
	if _, err := os.Stat(script); err != nil {
		fail(exitUnknown, "scripts/doctor.py not found under "+root)
		return exitUnknown
	}
	py := pythonBin()
	if py == "" {
		fail(exitUnknown, "python3 not on PATH")
		return exitUnknown
	}
	cmd := exec.Command(py, script, "--registry", *regPath, "--root", root)
	cmd.Stdout, cmd.Stderr, cmd.Env = os.Stdout, os.Stderr, os.Environ()
	if err := cmd.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		fail(exitInvalid, err.Error())
		return exitInvalid
	}
	return exitOK
}

func cmdGuard(args []string) int {
	fs := flag.NewFlagSet("guard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	toolName := fs.String("tool", "", "")
	payload := fs.String("payload", "", "")
	legacyCall := fs.String("call", "", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	raw := map[string]any{}
	switch {
	case *payload != "":
		if json.Unmarshal([]byte(*payload), &raw) != nil || raw == nil {
			fail(exitBadPayload, "parse --payload")
			return exitBadPayload
		}
	case *legacyCall != "":
		if json.Unmarshal([]byte(*legacyCall), &raw) != nil || raw == nil {
			fail(exitBadPayload, "parse --call")
			return exitBadPayload
		}
	}
	if _, ok := raw["tool"].(string); !ok || raw["tool"] == "" {
		if *toolName != "" {
			raw["tool"] = *toolName
		} else {
			fail(exitUsage, "missing --tool")
			return exitUsage
		}
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	res, err := guard.Validate(reg, raw)
	if err != nil {
		code := exitInvalid
		if strings.Contains(err.Error(), "unknown tool") {
			code = exitUnknown
		}
		fail(code, err.Error())
		return code
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"ok": true, "code": exitOK, "tool": res.Tool, "risk": res.Risk, "timezone": res.Timezone, "now": res.Now, "retries": res.Retries, "flat": res.Flat, "params": res.Params})
	return exitOK
}

func cmdGraph(args []string) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	vault := fs.String("vault", "", "")
	rootFlag := fs.String("root", "", "")
	query := fs.String("query", "", "")
	ext := fs.String("ext", ".md", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	path := *vault
	if path == "" {
		path = *rootFlag
	}
	if path == "" {
		path = os.Getenv("VAULT_PATH")
	}
	if path == "" {
		fail(exitUsage, "--vault is required")
		return exitUsage
	}
	g, err := graph.Scan(path, *ext)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(graph.Query(g, *query))
	return exitOK
}
