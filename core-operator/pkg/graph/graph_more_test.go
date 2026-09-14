package graph

import (
	"path/filepath"
	"testing"
)

func TestScanMissingRoot(t *testing.T) {
	if _, err := Scan(filepath.Join(t.TempDir(), "missing"), ".md"); err == nil {
		t.Fatal("expected missing vault error")
	}
}

func TestQueryEmptyAndNoMatch(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "A.md"), "hello")
	g, err := Scan(root, ".md")
	if err != nil {
		t.Fatal(err)
	}
	all := Query(g, "")
	if len(all.Nodes) != 1 {
		t.Fatalf("empty query should keep nodes: %+v", all.Nodes)
	}
	none := Query(g, "zzz-no-such")
	if len(none.Nodes) != 0 {
		t.Fatalf("expected no match: %+v", none.Nodes)
	}
}
