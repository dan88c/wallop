package pointer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidatePointers(t *testing.T) {
	tempDir := t.TempDir()

	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve current test executable: %v", err)
	}

	if err := RegisterTool(tempDir, ToolPointer{
		Name:        "test_valid",
		Description: "Self-referential valid tool",
		Command:     currentExe,
	}); err != nil {
		t.Fatalf("failed to register valid tool: %v", err)
	}

	toolsDir := filepath.Join(tempDir, "tools")
	if err := os.WriteFile(filepath.Join(toolsDir, "test_missing.json"), []byte(`{"name":"test_missing","command":"/nonexistent/binary"}`), 0644); err != nil {
		t.Fatalf("failed to write missing tool pointer: %v", err)
	}
	if err := os.WriteFile(filepath.Join(toolsDir, "../escape.json"), []byte(`{"name":"../escape","command":"`+currentExe+`"}`), 0644); err != nil {
		t.Fatalf("failed to write escaped name pointer: %v", err)
	}

	tools, err := LoadAndValidatePointers(tempDir)
	if err != nil {
		t.Fatalf("LoadAndValidatePointers failed: %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("expected exactly 1 valid tool, got %d", len(tools))
	}
	if tools[0].Name != "test_valid" {
		t.Errorf("expected 'test_valid', got '%s'", tools[0].Name)
	}
}

func TestRunHealthcheck(t *testing.T) {
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to resolve current test executable: %v", err)
	}

	ok := ToolPointer{Name: "ok", Command: currentExe, Healthcheck: []string{"-test.list", "TestRunHealthcheck"}}
	if err := RunHealthcheck(ok); err != nil {
		t.Fatalf("expected passing healthcheck, got %v", err)
	}

	bad := ToolPointer{Name: "bad", Command: currentExe, Healthcheck: []string{"-invalid-flag-that-fails"}}
	if err := RunHealthcheck(bad); err == nil {
		t.Fatal("expected failing healthcheck")
	}
}

func TestValidatePayload(t *testing.T) {
	schemaRaw := json.RawMessage(`{"required": ["query", "limit"]}`)

	validPayload := []byte(`{"query": "search notes", "limit": 10}`)
	if err := ValidatePayload(schemaRaw, validPayload); err != nil {
		t.Errorf("expected payload to be valid, got error: %v", err)
	}

	missingPayload := []byte(`{"query": "search notes"}`)
	if err := ValidatePayload(schemaRaw, missingPayload); err == nil {
		t.Errorf("expected validation failure for missing field 'limit', got nil")
	}

	malformedPayload := []byte(`{bad-json}`)
	if err := ValidatePayload(schemaRaw, malformedPayload); err == nil {
		t.Errorf("expected malformed JSON failure, got nil")
	}

	if err := ValidatePayload(json.RawMessage(`"not-an-object"`), validPayload); err == nil {
		t.Errorf("expected invalid schema to fail closed, got nil")
	}

	if err := ValidatePayload(nil, validPayload); err != nil {
		t.Errorf("empty schema should skip validation, got %v", err)
	}
}
