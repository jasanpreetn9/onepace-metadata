package util

import (
	"bytes"
	"os"
)

// Returns true if file exists *and* content is identical.
func FileUnchanged(path string, newData []byte) bool {
	oldData, err := os.ReadFile(path)
	if err != nil {
		return false // file missing → treat as changed
	}

	return bytes.Equal(bytes.TrimSpace(oldData), bytes.TrimSpace(newData))
}

// FileExists returns true if a file exists and is not a directory.
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false // file does not exist OR access error
	}
	return !info.IsDir()
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
