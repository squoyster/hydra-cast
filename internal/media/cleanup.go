package media

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func CleanupStaleFiles(dir string, maxAge time.Duration) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(dir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("remove stale file %s: %w", path, err)
			}
		}
	}

	return nil
}

func EnforceMaxWorkingBytes(dir string, maxBytes int64) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	type fileInfo struct {
		path    string
		size    int64
		modTime time.Time
	}

	var files []fileInfo
	var totalBytes int64

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, fileInfo{
			path:    filepath.Join(dir, entry.Name()),
			size:    info.Size(),
			modTime: info.ModTime(),
		})
		totalBytes += info.Size()
	}

	if totalBytes <= maxBytes {
		return nil
	}

	for i := len(files) - 1; i >= 0 && totalBytes > maxBytes; i-- {
		if err := os.Remove(files[i].path); err != nil {
			return fmt.Errorf("remove file %s: %w", files[i].path, err)
		}
		totalBytes -= files[i].size
	}

	return nil
}

func DeleteMedia(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete media %s: %w", path, err)
	}
	return nil
}
