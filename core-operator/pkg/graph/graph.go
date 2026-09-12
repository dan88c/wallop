package graph

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var wikiLink = regexp.MustCompile(`\[\[([^\]|#]+)(?:[|#][^\]]*)?\]\]`)
var hashTag = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])#([A-Za-z][A-Za-z0-9_/-]{1,40})`)

type Node struct {
	Path  string   `json:"path"`
	Title string   `json:"title"`
	Out   []string `json:"out"`
	In    []string `json:"in"`
	Tags  []string `json:"tags"`
}

type Graph struct {
	Root  string          `json:"root"`
	Nodes map[string]Node `json:"nodes"`
	Edges int             `json:"edges"`
}

func Scan(root, ext string) (*Graph, error) {
	if ext == "" {
		ext = ".md"
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	byTitle := map[string]string{}
	rawOut := map[string][]string{}
	rawTags := map[string][]string{}

	err = filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == ".git" || base == "node_modules" || base == ".obsidian" {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), strings.ToLower(ext)) {
			return nil
		}
		rel, _ := filepath.Rel(abs, path)
		title := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		byTitle[strings.ToLower(title)] = rel
		byTitle[strings.ToLower(rel)] = rel

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		matches := wikiLink.FindAllSubmatch(b, -1)
		seen := map[string]struct{}{}
		var outs []string
		for _, m := range matches {
			target := strings.TrimSpace(string(m[1]))
			key := strings.ToLower(target)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			outs = append(outs, target)
		}
		rawOut[rel] = outs
		tagSeen := map[string]struct{}{}
		var tags []string
		for _, m := range hashTag.FindAllSubmatch(b, -1) {
			tag := string(m[1])
			lk := strings.ToLower(tag)
			if _, ok := tagSeen[lk]; ok {
				continue
			}
			tagSeen[lk] = struct{}{}
			tags = append(tags, tag)
		}
		sort.Strings(tags)
		rawTags[rel] = tags
		return nil
	})
	if err != nil {
		return nil, err
	}

	g := &Graph{Root: abs, Nodes: map[string]Node{}}
	incoming := map[string][]string{}

	for rel, outs := range rawOut {
		resolved := make([]string, 0, len(outs))
		for _, t := range outs {
			if dest, ok := byTitle[strings.ToLower(t)]; ok {
				resolved = append(resolved, dest)
				incoming[dest] = append(incoming[dest], rel)
			} else {
				resolved = append(resolved, t+"?")
			}
		}
		sort.Strings(resolved)
		title := strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel))
		g.Nodes[rel] = Node{Path: rel, Title: title, Out: resolved, Tags: rawTags[rel]}
		g.Edges += len(resolved)
	}

	for rel, node := range g.Nodes {
		ins := incoming[rel]
		sort.Strings(ins)
		node.In = ins
		g.Nodes[rel] = node
	}
	return g, nil
}

func Query(g *Graph, keyword string) *Graph {
	if g == nil || strings.TrimSpace(keyword) == "" {
		return g
	}
	q := strings.ToLower(keyword)
	out := &Graph{Root: g.Root, Nodes: map[string]Node{}}
	for rel, n := range g.Nodes {
		if nodeMatches(n, q) {
			out.Nodes[rel] = n
			out.Edges += len(n.Out)
		}
	}
	return out
}

func nodeMatches(n Node, q string) bool {
	if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Path), q) {
		return true
	}
	for _, t := range n.Tags {
		if strings.Contains(strings.ToLower(t), q) {
			return true
		}
	}
	for _, l := range n.Out {
		if strings.Contains(strings.ToLower(l), q) {
			return true
		}
	}
	return false
}
