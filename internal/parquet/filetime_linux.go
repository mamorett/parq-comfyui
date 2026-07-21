//go:build linux

package parquet

import (
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// GetFileCreationTime returns the birth time (creation time) of a file on Linux
// using statx. If birth time is not supported by the filesystem/kernel,
// it falls back to the modification time.
func GetFileCreationTime(path string) (time.Time, error) {
	var statx unix.Statx_t
	err := unix.Statx(unix.AT_FDCWD, path, unix.AT_SYMLINK_NOFOLLOW, unix.STATX_BTIME, &statx)
	if err == nil {
		if (statx.Mask & unix.STATX_BTIME) != 0 {
			return time.Unix(statx.Btime.Sec, int64(statx.Btime.Nsec)), nil
		}
	}
	// Fallback to ModTime
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}, err
	}
	return info.ModTime(), nil
}
