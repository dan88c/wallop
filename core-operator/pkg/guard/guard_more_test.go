package guard

import (
	"testing"
)

func TestValidateMissingToolAndParam(t *testing.T) {
	reg := sampleReg()
	if _, err := Validate(reg, map[string]any{}); err == nil {
		t.Fatal("missing tool")
	}
	if _, err := Validate(reg, map[string]any{"tool": "nope", "action": "list"}); err == nil {
		t.Fatal("unknown tool")
	}
	if _, err := Validate(reg, map[string]any{"tool": "calendar_gateway"}); err == nil {
		t.Fatal("missing required action")
	}
}

func TestValidateListOK(t *testing.T) {
	reg := sampleReg()
	res, err := Validate(reg, map[string]any{"tool": "calendar_gateway", "action": "list"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK || res.Tool != "calendar_gateway" {
		t.Fatalf("%+v", res)
	}
}

func TestValidateBadTimezoneFallsBack(t *testing.T) {
	reg := sampleReg()
	reg.Timezone = "Not/A_Zone"
	res, err := Validate(reg, map[string]any{"tool": "calendar_gateway", "action": "list"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.OK {
		t.Fatal(res)
	}
}

func TestFlattenEmpty(t *testing.T) {
	if got := Flatten(nil); got == nil {
		t.Fatal("nil flatten")
	}
	if Flatten(map[string]any{})["x"] != nil {
		t.Fatal("empty map")
	}
}
