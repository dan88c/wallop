package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseParamSpecEmpty(t *testing.T) {
	if _, err := ParseParamSpec(""); err == nil {
		t.Fatal("expected empty param error")
	}
	if _, err := ParseParamSpec("   :string"); err == nil {
		t.Fatal("expected empty name error")
	}
	p, err := ParseParamSpec("q")
	if err != nil || p.Name != "q" || p.Type != "string" || p.Required {
		t.Fatalf("default param %+v %v", p, err)
	}
	p, err = ParseParamSpec("n:int:1")
	if err != nil || p.Type != "int" || !p.Required {
		t.Fatalf("required alias %+v %v", p, err)
	}
}

func TestSnakeFallback(t *testing.T) {
	if Snake("!!!") != "imported_tool" {
		t.Fatalf("got %s", Snake("!!!"))
	}
}

func TestCopyFileAndUpsertReplace(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.py")
	dst := filepath.Join(dir, "nested", "dst.py")
	if err := os.WriteFile(src, []byte("print(1)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != "print(1)\n" {
		t.Fatalf("copy failed %v %q", err, b)
	}

	reg := &Registry{Version: 1, Timezone: "Asia/Hong_Kong"}
	reg.Upsert(Tool{Name: "a", Desc: "one"})
	reg.Upsert(Tool{Name: "a", Desc: "two"})
	if len(reg.Tools) != 1 || reg.Tools[0].Desc != "two" {
		t.Fatalf("upsert replace: %+v", reg.Tools)
	}
	path := filepath.Join(dir, "reg.yaml")
	if err := reg.Save(path); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Timezone == "" || again.Tools[0].Desc != "two" {
		t.Fatalf("reload %+v", again)
	}
}

func TestFindUnknown(t *testing.T) {
	reg := &Registry{Tools: []Tool{{Name: "only"}}}
	if _, err := reg.Find("missing"); err == nil {
		t.Fatal("expected unknown tool")
	}
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Fatal("expected missing file")
	}
}

func TestRenderTOCTagFilter(t *testing.T) {
	reg := &Registry{
		Version:  1,
		Timezone: "Asia/Hong_Kong",
		Tools: []Tool{
			{Name: "keep", Tags: []string{"ops"}},
			{Name: "drop", Tags: []string{"demo"}},
		},
	}
	out, err := RenderTOC(reg, TOCOptions{Tag: "ops"})
	if err != nil {
		t.Fatal(err)
	}
	if !contains(out, "keep") || contains(out, "drop") {
		t.Fatalf("tag filter: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || (len(sub) > 0 && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()))
}
