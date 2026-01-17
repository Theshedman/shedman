package util

import (
	"io/fs"
	"os"
)

const (
	DirPermissions  fs.FileMode = 0755
	FilePermissions fs.FileMode = 0644
)

// FileExists checks if a file exists and is not a directory.
func FileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return !info.IsDir(), nil
}
