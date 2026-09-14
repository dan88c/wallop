package main

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func cmdRegister(args []string) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "")
	entry := fs.String("entry", "", "")
	name := fs.String("name", "", "")
	desc := fs.String("desc", "imported tool", "")
	details := fs.String("details", "", "")
	runtime := fs.String("runtime", "python", "")
	venv := fs.String("venv", "", "")
	risk := fs.String("risk", "read", "")
	keep := fs.Bool("keep-path", false, "")
	var tags stringList
	var params stringList
	fs.Var(&tags, "tag", "")
	fs.Var(&params, "param", "")
	if fs.Parse(args) != nil {
		return exitUsage
	}
	if *entry == "" || *name == "" {
		fail(exitUsage, "--entry and --name are required")
		return exitUsage
	}
	switch strings.ToLower(*risk) {
	case "read", "write":
	default:
		fail(exitUsage, "--risk must be read or write")
		return exitUsage
	}
	src, err := filepath.Abs(*entry)
	if err != nil {
		fail(exitInvalid, err.Error())
		return exitInvalid
	}
	if st, err := os.Stat(src); err != nil || st.IsDir() {
		fail(exitInvalid, "--entry must be an existing file: "+src)
		return exitInvalid
	}
	id := registry.Snake(*name)
	parsed := make([]registry.Param, 0, len(params))
	for _, spec := range params {
		p, err := registry.ParseParamSpec(spec)
		if err != nil {
			fail(exitUsage, err.Error())
			return exitUsage
		}
		parsed = append(parsed, p)
	}
	reg, err := registry.Load(*regPath)
	if err != nil {
		fail(exitUnknown, err.Error())
		return exitUnknown
	}
	root := registry.RepoRootFromRegistry(*regPath)
	catalogEntry := *entry
	copiedTo := ""
	if !*keep {
		destName := id + filepath.Ext(src)
		if destName == id {
			destName = id + ".py"
		}
		dest := filepath.Join(root, "tools", destName)
		if err := registry.CopyFile(src, dest); err != nil {
			fail(exitInvalid, "copy: "+err.Error())
			return exitInvalid
		}
		catalogEntry = filepath.ToSlash(filepath.Join("tools", destName))
		copiedTo = dest
	}
	tagList := []string(tags)
	if len(tagList) == 0 {
		tagList = []string{"imported"}
	}
	reg.Upsert(registry.Tool{Name: id, Desc: *desc, Details: *details, Tags: tagList, Entry: catalogEntry, Risk: strings.ToLower(*risk), Runtime: strings.ToLower(*runtime), Venv: *venv, Params: parsed})
	if err := reg.Save(*regPath); err != nil {
		fail(exitInvalid, err.Error())
		return exitInvalid
	}
	out := registry.ImportResult{Name: id, Entry: catalogEntry, CopiedTo: copiedTo, Registry: *regPath}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{"ok": true, "code": exitOK, "registered": out})
	return exitOK
}
