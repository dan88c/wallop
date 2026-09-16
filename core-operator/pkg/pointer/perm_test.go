package pointer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPermChecks(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Test IsStrictSecretPerm
	secretFile := filepath.Join(tempDir, "secrets.env")
	if err := os.WriteFile(secretFile, []byte("KEY=VALUE"), 0600); err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	info, err := os.Stat(secretFile)
	if err != nil {
		t.Fatalf("failed to stat secret file: %v", err)
	}

	if !IsStrictSecretPerm(info) {
		t.Errorf("expected 0600 file to pass strict secret perm check")
	}

	// Insecure permission test (only strictly verifiable on POSIX)
	if runtime.GOOS != "windows" {
		_ = os.Chmod(secretFile, 0644)
		insecureInfo, _ := os.Stat(secretFile)
		if IsStrictSecretPerm(insecureInfo) {
			t.Errorf("expected 0644 file to fail strict secret perm check on POSIX")
		}
	}

	// 2. Test IsExecutable
	currentExe, err := os.Executable()
	if err != nil {
		t.Fatalf("failed to get current executable: %v", err)
	}
	exeInfo, err := os.Stat(currentExe)
	if err != nil {
		t.Fatalf("failed to stat current executable: %v", err)
	}

	if !IsExecutable(exeInfo) {
		t.Errorf("expected running test binary to be executable")
	}
}