package pointer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateToolName(t *testing.T) {
	if err := ValidateToolName("echo_tool"); err != nil {
		t.Fatalf("valid name rejected: %v", err)
	}
	for _, name := range []string{"", "../evil", "/tmp/evil", "foo/bar", "foo.json"} {
		if err := ValidateToolName(name); err == nil {
			t.Errorf("expected invalid name %q to be rejected", name)
		}
	}
}

func TestPointerFileStaysInToolsDir(t *testing.T) {
	dir := t.TempDir()
	got, err := PointerFile(dir, "echo_tool")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "tools", "echo_tool.json")
	if got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	if _, err := PointerFile(dir, "../evil"); err == nil {
		t.Fatal("expected traversal name to be rejected")
	}
}

func TestRegisterToolRejectsUnsafeNameAndResolvesCommand(t *testing.T) {
	dir := t.TempDir()
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	if err := RegisterTool(dir, ToolPointer{Name: "../evil", Command: currentExe}); err == nil {
		t.Fatal("expected unsafe name to fail register")
	}

	if err := RegisterTool(dir, ToolPointer{Name: "ok_tool", Command: currentExe}); err != nil {
		t.Fatalf("expected register to succeed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "tools", "ok_tool.json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored ToolPointer
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Command != currentExe && !filepath.IsAbs(stored.Command) {
		t.Fatalf("expected resolved absolute command, got %s", stored.Command)
	}
}
