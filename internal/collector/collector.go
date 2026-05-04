package collector

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CollectImages collects PNG image paths based on input criteria
func CollectImages(input string, recursive bool, fileList string) ([]string, error) {
	var files []string

	// Priority 1: File list from text file
	if fileList != "" {
		f, err := os.Open(fileList)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			for _, path := range strings.Fields(line) {
				absPath, err := filepath.Abs(path)
				if err != nil {
					continue
				}
				if isSupportedImage(absPath) {
					files = append(files, absPath)
				}
			}
		}
		return sortAndUnique(files), scanner.Err()
	}

	// Priority 2: Input path (directory, file, or pattern)
	if input != "" {
		// Check if it's a pattern
		if strings.ContainsAny(input, "*?[]") {
			matches, err := filepath.Glob(input)
			if err != nil {
				return nil, err
			}
			for _, m := range matches {
				absPath, err := filepath.Abs(m)
				if err == nil && isSupportedImage(absPath) {
					files = append(files, absPath)
				}
			}
			return sortAndUnique(files), nil
		}

		info, err := os.Stat(input)
		if err != nil {
			return nil, err
		}

		if info.IsDir() {
			if recursive {
				err = filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}
					if !info.IsDir() && isSupportedImage(path) {
						absPath, err := filepath.Abs(path)
						if err == nil {
							files = append(files, absPath)
						}
					}
					return nil
				})
			} else {
				entries, err := os.ReadDir(input)
				if err != nil {
					return nil, err
				}
				for _, entry := range entries {
					if !entry.IsDir() && isSupportedImage(entry.Name()) {
						absPath, err := filepath.Abs(filepath.Join(input, entry.Name()))
						if err == nil {
							files = append(files, absPath)
						}
					}
				}
			}
			if err != nil {
				return nil, err
			}
		} else {
			if isSupportedImage(input) {
				absPath, err := filepath.Abs(input)
				if err == nil {
					files = append(files, absPath)
				}
			}
		}
	}

	return sortAndUnique(files), nil
}

func isSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".png"
}

func sortAndUnique(files []string) []string {
	if len(files) == 0 {
		return files
	}
	sort.Strings(files)
	unique := make([]string, 0, len(files))
	seen := make(map[string]struct{})
	for _, f := range files {
		if _, ok := seen[f]; !ok {
			unique = append(unique, f)
			seen[f] = struct{}{}
		}
	}
	return unique
}
