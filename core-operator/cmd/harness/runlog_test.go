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
	_ = os.WriteFile(reg, []byte("version: 1\ntimezone: Asia/Hong_Kong\ntools:\n  - name: echo_tool\n    desc: short\n    details: long\n    tags: [test]\n    entry: tools/echo_tool.py\n    risk: read\n    params:\n      - name: q\n        type: string\n        required: true\n"), 0o644)
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

func TestCmdPathDescRunMock(t *testing.T) {
	_, reg := mockReg(t)
	code, out := capJSON(t, func() int { return cmdPath([]string{"--registry", reg, "--tool", "echo_tool"}) })
	if code != 0 || out["tool"] != "echo_tool" {
		t.Fatalf("path %d %#v", code, out)
	}
	code, out = capJSON(t, func() int { return cmdDesc([]string{"--registry", reg, "--tool", "echo_tool"}) })
	if code != 0 || out["details"] != "long" {
		t.Fatalf("desc %d %#v", code, out)
	}
	runProcess = func(argv []string, cwd string, stdin []byte) processResult {
		return processResult{Stdout: ` + "`{\"ok\":true}`" + `}
	}
	code, out = capJSON(t, func() int {
		return cmdRun([]string{"--registry", reg, "--tool", "echo_tool", "--payload", ` + "`{\"q\":\"hi\"}`" + `})
	})
	if code != 0 {
		t.Fatalf("run ok %d %#v", code, out)
	}
	runProcess = func(argv []string, cwd string, stdin []byte) processResult {
		return processResult{Stderr: strings.Repeat("E", 3000), Err: errors.New("boom")}
	}
	code, out = capJSON(t, func() int {
		return cmdRun([]string{"--registry", reg, "--tool", "echo_tool", "--payload", ` + "`{\"q\":\"hi\"}`" + `})
	})
	if code != exitChild {
		t.Fatalf("want 5 got %d", code)
	}
	if !strings.Contains(out["stderr_head"].(string), "truncated") {
		t.Fatal("truncate")
	}
	called := false
	runProcess = func(argv []string, cwd string, stdin []byte) processResult {
		called = true
		return processResult{}
	}
	code, _ = capJSON(t, func() int {
		return cmdRun([]string{"--registry", reg, "--tool", "echo_tool", "--payload", ` + "`{}`" + `})
	})
	if code != exitInvalid || called {
		t.Fatalf("guard block code=%d called=%v", code, called)
	}
}
