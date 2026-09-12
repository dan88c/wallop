package guard

import (
	"fmt"
	"strings"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

const MaxRetries = 2

type Result struct {
	OK       bool           `json:"ok"`
	Tool     string         `json:"tool"`
	Risk     string         `json:"risk"`
	Timezone string         `json:"timezone"`
	Now      string         `json:"now"`
	Retries  int            `json:"retries"`
	Flat     map[string]any `json:"flat"`
	Params   map[string]any `json:"params"`
}

// Validate checks a proposed operator call against the registry and time rules.
// Write actions may not target a timestamp in the past (date-drift intercept).
func Validate(reg *registry.Registry, raw map[string]any) (*Result, error) {
	retries := asInt(raw["retry"])
	if retries == 0 {
		retries = asInt(raw["retries"])
	}
	if retries > MaxRetries {
		return nil, fmt.Errorf("retry budget exceeded: %d > %d", retries, MaxRetries)
	}

	name, _ := raw["tool"].(string)
	if name == "" {
		return nil, fmt.Errorf("missing tool")
	}
	tool, err := reg.Find(name)
	if err != nil {
		return nil, err
	}

	loc, err := time.LoadLocation(reg.Timezone)
	if err != nil {
		loc = time.FixedZone("HKT", 8*3600)
	}
	now := time.Now().In(loc)

	params := map[string]any{}
	for _, p := range tool.Params {
		v, ok := raw[p.Name]
		if p.Required && !ok {
			return nil, fmt.Errorf("missing required param %s", p.Name)
		}
		if !ok {
			continue
		}
		if err := typeCheck(p, v); err != nil {
			return nil, err
		}
		params[p.Name] = v

		if isTemporal(p.Type) {
			ts, err := parseTime(fmt.Sprint(v), loc)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", p.Name, err)
			}
			if isWrite(tool, raw) && ts.Before(now.Add(-2*time.Minute)) {
				return nil, fmt.Errorf("date-drift: %s=%s is in the past (now=%s tz=%s)",
					p.Name, ts.Format(time.RFC3339), now.Format(time.RFC3339), loc.String())
			}
		}
	}

	flat := Flatten(raw)
	return &Result{
		OK:       true,
		Tool:     tool.Name,
		Risk:     tool.Risk,
		Timezone: loc.String(),
		Now:      now.Format(time.RFC3339),
		Retries:  retries,
		Flat:     flat,
		Params:   params,
	}, nil
}

// Flatten turns nested maps into dotted keys so a 64k operator can emit one JSON object.
func Flatten(raw map[string]any) map[string]any {
	out := map[string]any{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		switch t := v.(type) {
		case map[string]any:
			for k, child := range t {
				next := k
				if prefix != "" {
					next = prefix + "." + k
				}
				walk(next, child)
			}
		default:
			if prefix != "" {
				out[prefix] = v
			}
		}
	}
	walk("", raw)
	return out
}

func asInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case string:
		n = strings.TrimSpace(n)
		var x int
		_, _ = fmt.Sscanf(n, "%d", &x)
		return x
	default:
		return 0
	}
}

func isWrite(tool *registry.Tool, raw map[string]any) bool {
	if strings.EqualFold(tool.Risk, "write") {
		return true
	}
	if a, ok := raw["action"].(string); ok {
		switch strings.ToLower(a) {
		case "create", "update", "delete", "write", "set":
			return true
		}
	}
	return false
}

func isTemporal(t string) bool {
	switch strings.ToLower(t) {
	case "datetime", "date", "time", "timestamp":
		return true
	}
	return false
}

func typeCheck(p registry.Param, v any) error {
	switch strings.ToLower(p.Type) {
	case "enum":
		s := fmt.Sprint(v)
		for _, allowed := range p.Values {
			if s == allowed {
				return nil
			}
		}
		return fmt.Errorf("%s=%v not in %v", p.Name, v, p.Values)
	case "string", "path":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("%s must be string", p.Name)
		}
	}
	return nil
}

func parseTime(s string, loc *time.Location) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
		if t, err := time.Parse(layout, s); err == nil {
			return t.In(loc), nil
		}
	}
	return time.Time{}, fmt.Errorf("unparseable time %q", s)
}
