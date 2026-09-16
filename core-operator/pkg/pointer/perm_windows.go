//go:build windows

package pointer

import (
	"os"
	"path/filepath"
	"strings"
)

// IsExecutable determines executability on Windows by common executable extensions.
func IsExecutable(info os.FileInfo) bool {
	ext := strings.ToLower(filepath.Ext(info.Name()))
	switch ext {
	case ".exe", ".bat", ".cmd", ".ps1", ".com":
		return true
	default:
		return false
	}
}

// IsStrictSecretPerm checks baseline file regularity on Windows NTFS.
func IsStrictSecretPerm(info os.FileInfo) bool {
	return info.Mode().IsRegular()
}