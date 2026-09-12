package registry

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Param struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	Required bool     `yaml:"required"`
	Values   []string `yaml:"values,omitempty"`
	Notes    string   `yaml:"notes,omitempty"`
}

type Tool struct {
	Name   string   `yaml:"name"`
	Desc   string   `yaml:"desc"`
	Tags   []string `yaml:"tags"`
	Entry  string   `yaml:"entry"`
	Risk   string   `yaml:"risk"`
	Params []Param  `yaml:"params"`
}

type Registry struct {
	Version    int    `yaml:"version"`
	Timezone   string `yaml:"timezone"`
	Tools      []Tool `yaml:"tools"`
	SourcePath string `yaml:"-"`
}

func Load(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry %s: %w", path, err)
	}
	var r Registry
	if err := yaml.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	if r.Timezone == "" {
		r.Timezone = "Asia/Hong_Kong"
	}
	r.SourcePath = path
	return &r, nil
}

func (r *Registry) Find(name string) (*Tool, error) {
	for i := range r.Tools {
		if r.Tools[i].Name == name {
			return &r.Tools[i], nil
		}
	}
	return nil, fmt.Errorf("unknown tool %q", name)
}

// DuplicateNames returns catalog names that appear more than once.
func (r *Registry) DuplicateNames() []string {
	seen := map[string]int{}
	for _, t := range r.Tools {
		if t.Name == "" {
			continue
		}
		seen[t.Name]++
	}
	var out []string
	for name, n := range seen {
		if n > 1 {
			out = append(out, name)
		}
	}
	return out
}

type TOCOptions struct {
	Full bool
	Tag  string
}

// RenderTOC is the progressive-disclosure surface for a weak operator model.
func RenderTOC(r *Registry, opt TOCOptions) (string, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "registry v%d tz=%s tools=%d\n", r.Version, r.Timezone, len(r.Tools))
	for _, t := range r.Tools {
		if opt.Tag != "" && !hasTag(t.Tags, opt.Tag) {
			continue
		}
		if opt.Full {
			fmt.Fprintf(&buf, "- %s [%s] risk=%s\n  %s\n  entry=%s\n",
				t.Name, strings.Join(t.Tags, ","), t.Risk, t.Desc, t.Entry)
			for _, p := range t.Params {
				req := ""
				if p.Required {
					req = "*"
				}
				extra := ""
				if len(p.Values) > 0 {
					extra = " values=" + strings.Join(p.Values, "|")
				}
				fmt.Fprintf(&buf, "  param %s%s type=%s%s\n", p.Name, req, p.Type, extra)
			}
			continue
		}
		fmt.Fprintf(&buf, "- %s [%s]\n", t.Name, strings.Join(t.Tags, ","))
	}
	return buf.String(), nil
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}
