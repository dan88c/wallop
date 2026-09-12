package registry

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var nonIdent = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func Snake(name string) string {
	s := strings.Trim(nonIdent.ReplaceAllString(name, "_"), "_")
	s = strings.ToLower(s)
	if s == "" {
		return "imported_tool"
	}
	return s
}

func ParseParamSpec(raw string) (Param, error) {
	bits := strings.Split(raw, ":")
	if len(bits) == 0 || strings.TrimSpace(bits[0]) == "" {
		return Param{}, fmt.Errorf("empty --param")
	}
	p := Param{Name: strings.TrimSpace(bits[0]), Type: "string"}
	if len(bits) > 1 && bits[1] != "" {
		p.Type = bits[1]
	}
	if len(bits) > 2 {
		switch strings.ToLower(bits[2]) {
		case "req", "required", "true", "1":
			p.Required = true
		}
	}
	return p, nil
}

func (r *Registry) Upsert(tool Tool) {
	for i := range r.Tools {
		if r.Tools[i].Name == tool.Name {
			r.Tools[i] = tool
			return
		}
	}
	r.Tools = append(r.Tools, tool)
}

func (r *Registry) Save(path string) error {
	if r.Version == 0 {
		r.Version = 1
	}
	if r.Timezone == "" {
		r.Timezone = "Asia/Hong_Kong"
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("write registry %s: %w", path, err)
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(r); err != nil {
		return fmt.Errorf("encode registry: %w", err)
	}
	return enc.Close()
}

func CopyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

// ImportResult is what wallop register prints after a successful catalog write.
type ImportResult struct {
	Name     string `json:"name"`
	Entry    string `json:"entry"`
	CopiedTo string `json:"copied_to,omitempty"`
	Registry string `json:"registry"`
}

func RepoRootFromRegistry(regPath string) string {
	abs, err := filepath.Abs(regPath)
	if err != nil {
		return filepath.Dir(filepath.Dir(regPath))
	}
	return filepath.Dir(filepath.Dir(abs))
}
