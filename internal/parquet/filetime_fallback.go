//go:build !windows && !darwin && !linux

package parquet

import (
	"os"
	"time"
)

// GetFileCreationTime returns the modification time of a file as a fallback
// for other systems where birth time is not standard or easily accessible.
func GetFileCreationTime(path string) (time.Time, error) {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
