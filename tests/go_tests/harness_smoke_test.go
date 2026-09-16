package go_tests

import (
	"os"
	"testing"

	"github.com/dan88c/wallop/core-operator/pkg/pointer"
)

func TestPointerCatalogRoundTrip(t *testing.T) {
	dir := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	if err := pointer.RegisterTool(dir, pointer.ToolPointer{
		Name:        "smoke_tool",
		Description: "Pointer catalog smoke tool",
		Command:     exe,
	}); err != nil {
		t.Fatal(err)
	}

	tools, err := pointer.LoadAndValidatePointers(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 || tools[0].Name != "smoke_tool" {
		t.Fatalf("expected smoke_tool in pointer catalog, got %+v", tools)
	}
}

func TestPointerNamesRejectPathEscape(t *testing.T) {
	if err := pointer.ValidateToolName("../evil"); err == nil {
		t.Fatal("expected path-like tool name to be rejected")
	}
	if _, err := pointer.PointerFile(t.TempDir(), "/tmp/evil"); err == nil {
		t.Fatal("expected absolute tool name to be rejected")
	}
}
