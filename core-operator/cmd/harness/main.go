package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dan88c/wallop/core-operator/pkg/graph"
	"github.com/dan88c/wallop/core-operator/pkg/guard"
	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

const (
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
	fmt.Fprint(os.Stderr, `wallop — typed operator shell

Commands:
  wallop toc [--full] [--tag name]
  wallop guard --tool <name> [--payload <json>]
  wallop graph --vault <path> [--query <keyword>]
  wallop register --entry <script.py> --name <id> [--desc ...] [--tag t] [--param name:type[:required]] [--risk read|write] [--keep-path]

Exit codes:
  0 ok
  1 invalid call (past date, bad enum, retry>2, missing param)
  2 usage
  3 unknown tool / registry
  4 unreadable payload
`)
}

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

func cmdTOC(args []string) int {
	fs := flag.NewFlagSet("toc", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "path to tool registry YAML")
	full := fs.Bool("full", false, "include descriptions, params, entry")
	tag := fs.String("tag", "", "filter by tag")
	if err := fs.Parse(args); err != nil {
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

func cmdGuard(args []string) int {
	fs := flag.NewFlagSet("guard", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "path to tool registry YAML")
	toolName := fs.String("tool", "", "registered tool name")
	payload := fs.String("payload", "", "flat JSON object of arguments")
	legacyCall := fs.String("call", "", "legacy: JSON including tool field")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}

	raw := map[string]any{}
	switch {
	case *payload != "":
		if err := json.Unmarshal([]byte(*payload), &raw); err != nil {
			fail(exitBadPayload, "parse --payload: "+err.Error())
			return exitBadPayload
		}
		if raw == nil {
			fail(exitBadPayload, "--payload must be a JSON object")
			return exitBadPayload
		}
	case *legacyCall != "":
		if err := json.Unmarshal([]byte(*legacyCall), &raw); err != nil {
			fail(exitBadPayload, "parse --call: "+err.Error())
			return exitBadPayload
		}
		if raw == nil {
			fail(exitBadPayload, "--call must be a JSON object")
			return exitBadPayload
		}
	default:
		raw = map[string]any{}
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
	_ = enc.Encode(map[string]any{
		"ok":       true,
		"code":     exitOK,
		"tool":     res.Tool,
		"risk":     res.Risk,
		"timezone": res.Timezone,
		"now":      res.Now,
		"retries":  res.Retries,
		"flat":     res.Flat,
		"params":   res.Params,
	})
	return exitOK
}

func cmdGraph(args []string) int {
	fs := flag.NewFlagSet("graph", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	vault := fs.String("vault", "", "markdown vault path")
	root := fs.String("root", "", "legacy alias of --vault")
	query := fs.String("query", "", "keyword over titles, tags, wikilinks")
	ext := fs.String("ext", ".md", "file extension to scan")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	path := *vault
	if path == "" {
		path = *root
	}
	if path == "" {
		if env := os.Getenv("VAULT_PATH"); env != "" {
			path = env
		}
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
	g = graph.Query(g, *query)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(g)
	return exitOK
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func cmdRegister(args []string) int {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	regPath := fs.String("registry", defaultRegistry(), "path to tool registry YAML")
	entry := fs.String("entry", "", "path to an existing Python script")
	name := fs.String("name", "", "catalog name (snake_case)")
	desc := fs.String("desc", "imported tool", "one-line description")
	risk := fs.String("risk", "read", "read or write")
	keep := fs.Bool("keep-path", false, "do not copy; store --entry as given")
	var tags stringList
	var params stringList
	fs.Var(&tags, "tag", "repeatable tag")
	fs.Var(&params, "param", "name:type[:required] (repeatable)")
	if err := fs.Parse(args); err != nil {
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

	reg.Upsert(registry.Tool{
		Name:   id,
		Desc:   *desc,
		Tags:   tagList,
		Entry:  catalogEntry,
		Risk:   strings.ToLower(*risk),
		Params: parsed,
	})
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

func fail(code int, msg string) {
	fmt.Fprintf(os.Stderr, "ERROR code=%d %s\n", code, msg)
	enc := json.NewEncoder(os.Stdout)
	_ = enc.Encode(map[string]any{"ok": false, "code": code, "error": msg})
}
