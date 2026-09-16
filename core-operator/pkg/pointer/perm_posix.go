//go:build !windows

package pointer

import "os"

// IsExecutable checks if the file has POSIX executable bits (+x).
func IsExecutable(info os.FileInfo) bool {
	return info.Mode().Perm()&0111 != 0
}

// IsStrictSecretPerm verifies that the file permissions are strictly 0600.
func IsStrictSecretPerm(info os.FileInfo) bool {
	return info.Mode().Perm() == 0600
}