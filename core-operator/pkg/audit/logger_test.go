package audit

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLogEvent(t *testing.T) {
	tempDir := t.TempDir()

	event1 := ExecutionEvent{
		Tool:       "search_notes",
		Status:     "success",
		ExitCode:   0,
		DurationMs: 42,
		Depth:      1,
	}

	event2 := ExecutionEvent{
		Tool:       "broken_tool",
		Status:     "failure",
		ExitCode:   1,
		DurationMs: 15,
		Depth:      2,
		Error:      "script terminated with exit code 1",
	}

	if err := LogEvent(tempDir, event1); err != nil {
		t.Fatalf("LogEvent 1 failed: %v", err)
	}
	if err := LogEvent(tempDir, event2); err != nil {
		t.Fatalf("LogEvent 2 failed: %v", err)
	}

	logPath := filepath.Join(tempDir, "wallop", "events.jsonl")
	file, err := os.Open(logPath)
	if err != nil {
		t.Fatalf("failed to open events.jsonl: %v", err)
	}
	defer file.Close()

	var recorded []ExecutionEvent
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var ev ExecutionEvent
		if err := json.Unmarshal(scanner.Bytes(), &ev); err != nil {
			t.Fatalf("failed to unmarshal JSONL line: %v", err)
		}
		recorded = append(recorded, ev)
	}

	if len(recorded) != 2 {
		t.Fatalf("expected 2 events recorded, got %d", len(recorded))
	}
	if recorded[0].Tool != "search_notes" || recorded[0].ExitCode != 0 {
		t.Errorf("event 1 corrupted: %+v", recorded[0])
	}
	if recorded[1].Tool != "broken_tool" || recorded[1].Status != "failure" {
		t.Errorf("event 2 corrupted: %+v", recorded[1])
	}
}