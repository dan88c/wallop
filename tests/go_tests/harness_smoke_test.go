package go_tests

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

func TestRepoRegistryLoads(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "config", "tool_registry.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	reg, err := registry.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Find("calendar_gateway"); err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Find("time_ops_reader"); err != nil {
		t.Fatal(err)
	}
}
