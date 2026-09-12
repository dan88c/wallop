package guard

import (
	"testing"
	"time"

	"github.com/dan88c/wallop/core-operator/pkg/registry"
)

func sampleReg() *registry.Registry {
	return &registry.Registry{
		Version:  1,
		Timezone: "Asia/Hong_Kong",
		Tools: []registry.Tool{
			{
				Name: "calendar_gateway",
				Risk: "write",
				Params: []registry.Param{
					{Name: "action", Type: "enum", Required: true, Values: []string{"list", "create", "update", "delete"}},
					{Name: "start", Type: "datetime", Required: false},
				},
			},
		},
	}
}

func TestValidateRejectsPastWrite(t *testing.T) {
	reg := sampleReg()
	_, err := Validate(reg, map[string]any{
		"tool":   "calendar_gateway",
		"action": "create",
		"start":  "2020-01-01T00:00:00+08:00",
	})
	if err == nil || !contains(err.Error(), "date-drift") {
		t.Fatalf("expected date-drift, got %v", err)
	}
}

func TestValidateAllowsFutureWrite(t *testing.T) {
	reg := sampleReg()
	future := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
	res, err := Validate(reg, map[string]any{
		"tool":   "calendar_gateway",
		"action": "create",
		"start":  future,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal("expected ok")
	}
}

func TestValidateRejectsRetryOverBudget(t *testing.T) {
	reg := sampleReg()
	_, err := Validate(reg, map[string]any{
		"tool":   "calendar_gateway",
		"action": "list",
		"retry":  3,
	})
	if err == nil {
		t.Fatal("expected retry budget error")
	}
}

func TestFlattenDottedKeys(t *testing.T) {
	flat := Flatten(map[string]any{"a": map[string]any{"b": 1}})
	if flat["a.b"] != 1 {
		t.Fatalf("got %#v", flat)
	}
}

func TestValidateRejectsUnknownEnum(t *testing.T) {
	reg := sampleReg()
	_, err := Validate(reg, map[string]any{
		"tool":   "calendar_gateway",
		"action": "explode",
	})
	if err == nil {
		t.Fatal("expected enum error")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})())
}
