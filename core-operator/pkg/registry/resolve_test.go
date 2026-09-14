package registry

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveEntryRelativeAndAbsolute(t *testing.T) {
	root := t.TempDir()
	rel := ResolveEntry(root, "tools/x.py")
	want := filepath.Join(root, "tools", "x.py")
	if rel != want {
		t.Fatalf("rel: got %s want %s", rel, want)
	}
	abs := filepath.Join(root, "outside.py")
	if got := ResolveEntry(root, abs); got != abs {
		t.Fatalf("abs: got %s want %s", got, abs)
	}
	if ResolveEntry(root, "") != "" {
		t.Fatal("empty entry should be empty")
	}
	if ResolveEntry(root, "  tools/a.py  ") != filepath.Join(root, "tools", "a.py") {
		t.Fatal("trim failed")
	}
}

func TestResolveEntryHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip(err)
	}
	got := ResolveEntry("/repo", "~/tools/x.py")
	if !strings.HasPrefix(got, home) {
		t.Fatalf("expected home prefix %s got %s", home, got)
	}
}

func TestResolveVenvOrder(t *testing.T) {
	root := t.TempDir()
	perTool := filepath.Join(root, "toolvenv")
	if got := ResolveVenv(root, perTool); got != perTool {
		t.Fatalf("tool venv wins: %s", got)
	}
	t.Setenv("WALLOP_VENV", filepath.Join(root, "envvenv"))
	if got := ResolveVenv(root, ""); got != filepath.Join(root, "envvenv") {
		t.Fatalf("env venv: %s", got)
	}
	t.Setenv("WALLOP_VENV", "")
	if got := ResolveVenv(root, ""); got != filepath.Join(root, ".venv") {
		t.Fatalf("default venv: %s", got)
	}
}

func TestVenvPythonOS(t *testing.T) {
	got := VenvPython("/opt/venv")
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(got, `Scripts\python.exe`) && !strings.Contains(got, "Scripts") {
			t.Fatalf("windows python: %s", got)
		}
		return
	}
	if got != "/opt/venv/bin/python" && !strings.HasSuffix(got, "bin/python") {
		t.Fatalf("posix python: %s", got)
	}
}

func TestResolveCommandByOS(t *testing.T) {
	tool := Tool{
		Runtime: "shell",
		Command: OSCommand{Windows: "rg.exe", Posix: "rg"},
		Entry:   "rg",
	}
	got := ResolveCommand(tool)
	if runtime.GOOS == "windows" {
		if got != "rg.exe" {
			t.Fatalf("want rg.exe got %s", got)
		}
	} else if got != "rg" {
		t.Fatalf("want rg got %s", got)
	}
}

func TestRuntimeOfDefault(t *testing.T) {
	if RuntimeOf(Tool{}) != "python" {
		t.Fatal("default runtime")
	}
	if RuntimeOf(Tool{Runtime: " SHELL "}) != "shell" {
		t.Fatal("normalize runtime")
	}
}

func TestRepoRootFromRegistry(t *testing.T) {
	dir := t.TempDir()
	reg := filepath.Join(dir, "config", "tool_registry.yaml")
	if err := os.MkdirAll(filepath.Dir(reg), 0o755); err != nil {
		t.Fatal(err)
	}
	root := RepoRootFromRegistry(reg)
	if root != dir && filepath.Clean(root) != filepath.Clean(dir) {
		t.Fatalf("root=%s dir=%s", root, dir)
	}
}
