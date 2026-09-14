package main

import (
	"flag"
	"os"

	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

func cmdPath(args []string) int {
	fs := flag.NewFlagSet("path", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	toolName := fs.String("tool", "", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	name := *toolName
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if name == "" {
		fail(exitUsage, "missing --tool")
		return exitUsage
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	tool, err := reg.Find(name)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	root := registry.RepoRootFromRegistry(*regPath)
	resolved := registry.ResolveEntry(root, tool.Entry)
	ok := fileExists(resolved) || registry.RuntimeOf(*tool) != "python"
	code := exitOK
	if !ok {
		code = exitInvalid
	}
	printJSON(map[string]any{"ok": ok, "code": code, "tool": tool.Name, "runtime": registry.RuntimeOf(*tool), "entry": tool.Entry, "path": resolved, "exists": fileExists(resolved), "venv": registry.ResolveVenv(root, tool.Venv), "python": registry.VenvPython(registry.ResolveVenv(root, tool.Venv))})
	return code
}

func cmdDesc(args []string) int {
	fs := flag.NewFlagSet("desc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	toolName := fs.String("tool", "", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	name := *toolName
	if name == "" && fs.NArg() > 0 {
		name = fs.Arg(0)
	}
	if name == "" {
		fail(exitUsage, "missing --tool")
		return exitUsage
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	tool, err := reg.Find(name)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	details := tool.Details
	if details == "" {
		details = tool.Desc
	}
	root := registry.RepoRootFromRegistry(*regPath)
	printJSON(map[string]any{"ok": true, "code": 0, "tool": tool.Name, "desc": tool.Desc, "details": details, "params": tool.Params, "path": registry.ResolveEntry(root, tool.Entry), "venv": registry.ResolveVenv(root, tool.Venv), "runtime": registry.RuntimeOf(*tool)})
	return exitOK
}
