//go:build darwin

package parquet

import (
	"os"
	"syscall"
	"time"
)

// GetFileCreationTime returns the birth time (creation time) of a file on Darwin.
// It falls back to modification time if retrieval fails.
func GetFileCreationTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	if d, ok := info.Sys().(*syscall.Stat_t); ok {
		return time.Unix(d.Birthtimespec.Sec, d.Birthtimespec.Nsec), nil
	}
	return info.ModTime(), nil
}
