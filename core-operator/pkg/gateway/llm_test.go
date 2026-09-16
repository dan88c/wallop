package gateway

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadSecretsPermissions(t *testing.T) {
	tempDir := t.TempDir()
	secretPath := filepath.Join(tempDir, "secrets.env")

	// 1. Insecure file permission (0644)
	if err := os.WriteFile(secretPath, []byte("CLOUD_DEVELOPER_TOKEN=test-token\n"), 0644); err != nil {
		t.Fatalf("failed to write secrets file: %v", err)
	}

	_, err := LoadSecrets(tempDir)
	if runtime.GOOS != "windows" {
		if err == nil {
			t.Errorf("expected error due to insecure permissions (0644), got nil")
		}
		if err := os.Chmod(secretPath, 0600); err != nil {
			t.Fatalf("failed to chmod secrets file: %v", err)
		}
	} else if err != nil {
		t.Fatalf("expected Windows regular secrets.env to load, got %v", err)
	}

	secrets, err := LoadSecrets(tempDir)
	if err != nil {
		t.Fatalf("expected successful read for 0600 permissions, got error: %v", err)
	}

	if secrets["CLOUD_DEVELOPER_TOKEN"] != "test-token" {
		t.Errorf("expected token 'test-token', got '%s'", secrets["CLOUD_DEVELOPER_TOKEN"])
	}
}

func TestWriteSynthesizedToolSafely(t *testing.T) {
	tempDir := t.TempDir()
	scriptContent := "#!/usr/bin/env python3\nprint('synthesized tool execution')"

	scriptPath, err := WriteSynthesizedToolSafely(tempDir, scriptContent)
	if err != nil {
		t.Fatalf("WriteSynthesizedToolSafely failed: %v", err)
	}

	// 1. Verify file existence and readability
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("failed to read synthesized tool at %s: %v", scriptPath, err)
	}
	if string(content) != scriptContent {
		t.Errorf("script content mismatch: got %s, expected %s", string(content), scriptContent)
	}

	// 2. Verify executable permission (0755)
	info, err := os.Stat(scriptPath)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Errorf("expected synthesized tool to have executable permissions (+x), got %04o", info.Mode().Perm())
	}

	// 3. Verify that no temporary file (.tmp) was left behind
	tmpFiles, _ := filepath.Glob(filepath.Join(tempDir, "generated", "*.tmp"))
	if len(tmpFiles) > 0 {
		t.Errorf("temporary file was not cleaned up after atomic commit: %v", tmpFiles)
	}
}
