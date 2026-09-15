package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const exitChild = 5

type processResult struct {
	Stdout, Stderr string
	Err            error
}

type processRunner func(argv []string, cwd string, stdin []byte) processResult

var runProcess processRunner = func(argv []string, cwd string, stdin []byte) processResult {
	if len(argv) == 0 {
		return processResult{Err: errors.New("empty argv")}
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Stdin = bytes.NewReader(stdin)
	var o, e bytes.Buffer
	cmd.Stdout, cmd.Stderr = &o, &e
	return processResult{Stdout: o.String(), Stderr: e.String(), Err: cmd.Run()}
}

var lookPathFn = exec.LookPath

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n...[truncated, see log]\n"
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func flagsFromPayload(raw map[string]any) []string {
	var keys []string
	for k := range raw {
		if k != "tool" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	var out []string
	for _, k := range keys {
		if raw[k] == nil {
			continue
		}
		out = append(out, "--"+k, fmt.Sprint(raw[k]))
	}
	return out
}

func writeRunLog(root, tool string, code int, argv []string, payload, stdout, stderr string) string {
	dir := filepath.Join(root, "logs")
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, time.Now().Format("20060102T150405")+"_"+tool+".exit"+fmt.Sprint(code)+".log")
	var b strings.Builder
	fmt.Fprintf(&b, "tool=%s code=%d\nargv=%s\npayload=%s\n--- stdout ---\n%s\n--- stderr ---\n%s\n",
		tool, code, strings.Join(argv, " "), clip(payload, 8192), stdout, stderr)
	_ = os.WriteFile(path, []byte(b.String()), 0o644)
	return path
}
