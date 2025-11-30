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
