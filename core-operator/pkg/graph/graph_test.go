package graph

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanBidirectionalLinks(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "Home.md"), "See [[Projects]] and [[Home]].\n#demo #ops")
	mustWrite(t, filepath.Join(root, "Projects.md"), "Back to [[Home]].")

	g, err := Scan(root, ".md")
	if err != nil {
		t.Fatal(err)
	}
	home := g.Nodes["Home.md"]
	if len(home.Out) == 0 {
		t.Fatalf("expected outbound links, got %+v", home)
	}
	proj := g.Nodes["Projects.md"]
	found := false
	for _, in := range proj.In {
		if in == "Home.md" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected inbound Home.md on Projects.md, got %+v", proj)
	}
	if len(home.Tags) == 0 {
		t.Fatalf("expected hash tags on Home.md")
	}
	q := Query(g, "demo")
	if _, ok := q.Nodes["Home.md"]; !ok {
		t.Fatalf("query demo missed Home.md: %+v", q.Nodes)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
