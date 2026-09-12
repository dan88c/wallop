package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderTOCCompressedAndFull(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reg.yaml")
	yaml := `
version: 1
timezone: Asia/Hong_Kong
tools:
  - name: calendar_gateway
    desc: Call the edge calendar webhook.
    tags: [calendar, write]
    entry: tools/calendar_gateway.py
    risk: write
    params:
      - name: action
        type: enum
        required: true
        values: [list, create]
`
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := RenderTOC(reg, TOCOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compact, "Call the edge") {
		t.Fatalf("compressed TOC leaked description: %s", compact)
	}
	if !strings.Contains(compact, "calendar_gateway") {
		t.Fatalf("missing tool name: %s", compact)
	}
	full, err := RenderTOC(reg, TOCOptions{Full: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(full, "param action*") {
		t.Fatalf("full TOC missing required param: %s", full)
	}
}
