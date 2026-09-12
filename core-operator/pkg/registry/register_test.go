package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnakeAndParamSpec(t *testing.T) {
	if Snake("My Counter") != "my_counter" {
		t.Fatalf("snake: %s", Snake("My Counter"))
	}
	p, err := ParseParamSpec("text:string:required")
	if err != nil || p.Name != "text" || p.Type != "string" || !p.Required {
		t.Fatalf("param: %+v %v", p, err)
	}
}

func TestUpsertAndSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "reg.yaml")
	if err := os.WriteFile(path, []byte("version: 1\ntimezone: Asia/Hong_Kong\ntools: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	reg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	reg.Upsert(Tool{Name: "my_counter", Desc: "count", Tags: []string{"text"}, Entry: "tools/my_counter.py", Risk: "read"})
	if err := reg.Save(path); err != nil {
		t.Fatal(err)
	}
	again, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := again.Find("my_counter"); err != nil {
		t.Fatal(err)
	}
}
