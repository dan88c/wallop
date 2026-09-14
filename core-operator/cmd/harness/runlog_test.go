package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mockReg(t *testing.T) (root, reg string) {
	t.Helper()
	root = t.TempDir()
	_ = os.MkdirAll(filepath.Join(root, "config"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, "tools"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "tools", "echo_tool.py"), []byte("x\n"), 0o644)
	reg = filepath.Join(root, "config", "tool_registry.yaml")
	_ = os.WriteFile(reg, []byte(`version: 1
timezone: Asia/Hong_Kong
tools:
  - name: echo_tool
    desc: short
    details: long
    tags: [test]
    entry: tools/echo_tool.py
    risk: read
    params:
      - name: q
        type: string
        required: true
`), 0o644)
	return root, reg
}

func capJSON(t *testing.T, fn func() int) (int, map[string]any) {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	code := fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := map[string]any{}
	s := buf.String()
	if i := strings.Index(s, "{"); i >= 0 {
		_ = json.Unmarshal([]byte(s[i:]), &out)
	}
	return code, out
}

func TestFlagsAndClip(t *testing.T) {
	got := strings.Join(flagsFromPayload(map[string]any{"tool": "x", "q": "a"}), " ")
	if strings.Contains(got, "--tool") || !strings.Contains(got, "--q a") {
		t.Fatalf("%s", got)
	}
	if !strings.Contains(clip(strings.Repeat("e", 30), 8), "truncated") {
		t.Fatal("clip")
	}
}
