package util

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/theshedman/shedman/pkg/executor"
)

func GetPrivilegedRcloneCommand(args []string) []string {
	if os.Geteuid() != 0 {
		out, err := (&executor.RealExecutor{}).Output("rclone", "config", "file")
		if err == nil {

			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) > 0 {
				configPath := strings.TrimSpace(lines[len(lines)-1])
				if strings.HasPrefix(configPath, "/") {
					newArgs := append([]string{"--config", configPath}, args...)
					return append([]string{"sudo", "-E", "rclone"}, newArgs...)
				}
			}
		}
		return append([]string{"rclone"}, args...)
	}

	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		// Validate username to prevent path traversal attacks
		// e.g., SUDO_USER="../../../etc" would be rejected
		if err := ValidateUsername(sudoUser); err == nil {
			userConfig := filepath.Join("/home", sudoUser, ".config/rclone/rclone.conf")
			if _, err := os.Stat(userConfig); err == nil {
				return append([]string{"rclone", "--config", userConfig}, args...)
			}
		}
		// If validation fails, fall through to default behavior
	}

	return append([]string{"rclone"}, args...)
}
