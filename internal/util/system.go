package util

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// IsCommandAvailable checks if a command exists in PATH
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetRootFSType returns the filesystem type of the root partition (/)
func GetRootFSType() string {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return "unknown"
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 3 {
			mountPoint := fields[1]
			fsType := fields[2]
			if mountPoint == "/" {
				return fsType
			}
		}
	}
	return "unknown"
}
