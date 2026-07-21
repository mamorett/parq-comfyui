//go:build windows

package parquet

import (
	"os"
	"syscall"
	"time"
)

// GetFileCreationTime returns the creation time of a file on Windows.
// It falls back to modification time if retrieval fails.
func GetFileCreationTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	if d, ok := info.Sys().(*syscall.Win32FileAttributeData); ok {
		return time.Unix(0, d.CreationTime.Nanoseconds()), nil
	}
	return info.ModTime(), nil
}
