package doctor

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunDiagnostics_PermissionsAndWritable(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	stateDir := filepath.Join(tempDir, "state")
	_ = os.MkdirAll(configDir, 0755)
	_ = os.MkdirAll(stateDir, 0755)

	// 1. POSIX-only test for insecure permissions
	if runtime.GOOS != "windows" {
		secretPath := filepath.Join(configDir, "secrets.env")
		if err := os.WriteFile(secretPath, []byte("API_KEY=test"), 0644); err != nil {
			t.Fatalf("failed to write insecure secrets.env: %v", err)
		}

		var buf bytes.Buffer
		err := RunDiagnosticsWithWriter(&buf, configDir, stateDir)

		output := buf.String()
		if err == nil {
			t.Errorf("expected RunDiagnosticsWithWriter to return error on 0644 secrets.env")
		}
		if !strings.Contains(output, "[FAIL] secrets.env permissions are insecure") {
			t.Errorf("expected fail log message for secrets permission, got: %s", output)
		}

		_ = os.Remove(secretPath)
	}

	// 2. Directory write tests
	var buf bytes.Buffer
	_ = RunDiagnosticsWithWriter(&buf, configDir, stateDir)
	output := buf.String()

	if !strings.Contains(output, "[PASS] State Directory is writable") {
		t.Errorf("expected state directory to pass writable check, got: %s", output)
	}
	if !strings.Contains(output, "[PASS] Tools Directory is writable") {
		t.Errorf("expected tools directory to pass writable check, got: %s", output)
	}
}