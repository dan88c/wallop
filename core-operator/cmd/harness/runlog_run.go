package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dan88c/wallop/core-operator/pkg/guard"
	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

func cmdRun(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	toolName := fs.String("tool", "", "")
	payloadRaw := fs.String("payload", "{}", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	if *toolName == "" {
		fail(exitUsage, "missing --tool")
		return exitUsage
	}
	raw := map[string]any{}
	if json.Unmarshal([]byte(*payloadRaw), &raw) != nil || raw == nil {
		fail(exitBadPayload, "parse --payload")
		return exitBadPayload
	}
	raw["tool"] = *toolName
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	if _, err := guard.Validate(reg, raw); err != nil {
		code := exitInvalid
		if strings.Contains(err.Error(), "unknown tool") {
			code = exitUnknown
		}
		fail(code, err.Error())
		return code
	}
	tool, _ := reg.Find(*toolName)
	root := registry.RepoRootFromRegistry(*regPath)
	rt := registry.RuntimeOf(*tool)
	var argv []string
	switch rt {
	case "python":
		py := registry.VenvPython(registry.ResolveVenv(root, tool.Venv))
		if !fileExists(py) {
			if fb := pythonBin(); fb != "" {
				py = fb
			}
		}
		entry := registry.ResolveEntry(root, tool.Entry)
		if !fileExists(entry) {
			fail(exitInvalid, "entry does not exist: "+entry)
			return exitInvalid
		}
		argv = append([]string{py, entry}, flagsFromPayload(raw)...)
	case "shell", "exec":
		cmdPath := registry.ResolveCommand(*tool)
		if cmdPath == "" {
			fail(exitInvalid, "no command for this OS")
			return exitInvalid
		}
		if filepath.IsAbs(cmdPath) {
			if !fileExists(cmdPath) {
				fail(exitInvalid, "command not found: "+cmdPath)
				return exitInvalid
			}
		} else if p, err := lookPathFn(cmdPath); err != nil {
			fail(exitInvalid, "command not found: "+cmdPath)
			return exitInvalid
		} else {
			cmdPath = p
		}
		argv = append([]string{cmdPath}, tool.Argv...)
		argv = append(argv, flagsFromPayload(raw)...)
	default:
		fail(exitInvalid, "unknown runtime "+rt)
		return exitInvalid
	}
	stdin, _ := json.Marshal(raw)
	res := runProcess(argv, root, stdin)
	code := exitOK
	if res.Err != nil {
		code = exitChild
	}
	logPath := writeRunLog(root, tool.Name, code, argv, *payloadRaw, res.Stdout, res.Stderr)
	printJSON(map[string]any{"ok": code == 0, "code": code, "tool": tool.Name, "argv": argv, "log": logPath, "stdout_head": clip(res.Stdout, 2048), "stderr_head": clip(res.Stderr, 2048)})
	return code
}

func cmdLog(args []string) int {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	toolName := fs.String("tool", "", "")
	_ = fs.Bool("last", true, "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	root := registry.RepoRootFromRegistry(*regPath)
	matches, _ := filepath.Glob(filepath.Join(root, "logs", "*.log"))
	sort.Strings(matches)
	if *toolName != "" {
		var f []string
		for _, m := range matches {
			if strings.Contains(filepath.Base(m), "_"+*toolName+".") {
				f = append(f, m)
			}
		}
		matches = f
	}
	if len(matches) == 0 {
		fail(exitUnknown, "no run logs")
		return exitUnknown
	}
	target := matches[len(matches)-1]
	b, err := os.ReadFile(target)
	if err != nil {
		fail(exitInvalid, err.Error())
		return exitInvalid
	}
	fmt.Printf("log=%s\n", target)
	_, _ = os.Stdout.Write(b)
	return exitOK
}
