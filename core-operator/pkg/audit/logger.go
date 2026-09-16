package audit

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ExecutionEvent struct {
	RunID          string `json:"run_id"`
	Timestamp      string `json:"timestamp"`
	Tool           string `json:"tool"`
	Status         string `json:"status"`
	ExitCode       int    `json:"exit_code"`
	DurationMs     int64  `json:"duration_ms"`
	Depth          int    `json:"depth"`
	PayloadSummary string `json:"payload_summary,omitempty"`
	StderrSnippet  string `json:"stderr_snippet,omitempty"`
	Error          string `json:"error,omitempty"`
}

func GenerateRunID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func LogEvent(stateDir string, event ExecutionEvent) error {
	logDir := filepath.Join(stateDir, "wallop")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	logFile := filepath.Join(logDir, "events.jsonl")
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open events.jsonl: %w", err)
	}
	defer f.Close()

	if event.Timestamp == "" {
		event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if event.RunID == "" {
		event.RunID = GenerateRunID()
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}