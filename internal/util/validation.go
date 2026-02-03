package util

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Username validation pattern: Unix username rules (relaxed for compatibility)
// Must start with letter or underscore, followed by letters, digits, underscores, or hyphens
// Note: POSIX is stricter (lowercase only), but many systems allow uppercase
var usernameRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_-]*$`)

// ZFS dataset name validation pattern
// Alphanumeric, underscores, slashes (path separator), hyphens, dots, and @ (snapshot separator)
var zfsDatasetRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_/\-.@]*$`)

// ValidateUsername validates a Unix username.
// Returns an error if the username is invalid or could be used for injection.
func ValidateUsername(username string) error {
	if username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if len(username) > 32 {
		return fmt.Errorf("username too long (max 32 characters)")
	}
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("invalid username: must start with letter or underscore, contain only letters, digits, underscores, and hyphens")
	}
	return nil
}

// ValidateZFSDatasetName validates a ZFS dataset or snapshot name.
// Returns an error if the name contains invalid characters.
func ValidateZFSDatasetName(name string) error {
	if name == "" {
		return fmt.Errorf("dataset name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("dataset name too long (max 255 characters)")
	}
	if !zfsDatasetRegex.MatchString(name) {
		return fmt.Errorf("invalid ZFS dataset name: contains illegal characters")
	}
	// Ensure no double slashes, leading/trailing slashes
	if strings.Contains(name, "//") {
		return fmt.Errorf("invalid ZFS dataset name: contains double slashes")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("invalid ZFS dataset name: cannot start or end with slash")
	}
	return nil
}

// ValidateHostname validates a hostname or IP address for SSH connections.
// Returns an error if the host is invalid or could be used for injection.
func ValidateHostname(host string) error {
	if host == "" {
		return fmt.Errorf("hostname cannot be empty")
	}
	if len(host) > 253 {
		return fmt.Errorf("hostname too long")
	}

	// Check for command injection characters
	if strings.ContainsAny(host, ";&|`$(){}[]<>!#'\"\\") {
		return fmt.Errorf("hostname contains invalid characters")
	}

	// Try parsing as IP first
	if ip := net.ParseIP(host); ip != nil {
		return nil
	}

	// Validate as hostname
	// Allow user@host format
	if idx := strings.LastIndex(host, "@"); idx >= 0 {
		user := host[:idx]
		hostname := host[idx+1:]
		if user != "" {
			if err := ValidateUsername(user); err != nil {
				return fmt.Errorf("invalid user in host: %w", err)
			}
		}
		host = hostname
	}

	// Basic hostname validation (RFC 1123)
	parts := strings.Split(host, ".")
	for _, part := range parts {
		if len(part) == 0 || len(part) > 63 {
			return fmt.Errorf("invalid hostname: label length")
		}
		for i, c := range part {
			isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
			isDigit := c >= '0' && c <= '9'
			isHyphen := c == '-' && i > 0 && i < len(part)-1
			if !isLetter && !isDigit && !isHyphen {
				return fmt.Errorf("invalid hostname: illegal character '%c'", c)
			}
		}
	}
	return nil
}

// ValidateExecutablePath validates that a path points to an executable file.
// Returns an error if the path is not a valid executable.
func ValidateExecutablePath(path string) error {
	if path == "" {
		return fmt.Errorf("executable path cannot be empty")
	}

	// Check for shell metacharacters that could indicate command injection
	if strings.ContainsAny(path, ";&|`$(){}[]<>!#'\"\\") {
		return fmt.Errorf("executable path contains shell metacharacters")
	}

	// If it's a bare command name (no path separators), look it up in PATH
	if !strings.Contains(path, string(os.PathSeparator)) {
		resolved, err := exec.LookPath(path)
		if err != nil {
			return fmt.Errorf("command not found in PATH: %s", path)
		}
		path = resolved
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}

	// Check file exists and is executable
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("executable not found: %s", absPath)
		}
		return fmt.Errorf("cannot access executable: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("path is a directory, not an executable: %s", absPath)
	}

	// Check executable permission
	if info.Mode()&0111 == 0 {
		return fmt.Errorf("file is not executable: %s", absPath)
	}

	return nil
}

// SanitizePath ensures a path component doesn't contain path traversal attempts.
// Returns an error if the path contains directory separators or traversal patterns.
func SanitizePath(component string) error {
	if component == "" {
		return fmt.Errorf("path component cannot be empty")
	}
	if strings.ContainsAny(component, `/\`) {
		return fmt.Errorf("path component contains directory separators")
	}
	if component == "." || component == ".." {
		return fmt.Errorf("path component is a traversal pattern")
	}
	// Check for .. anywhere in the string (traversal pattern)
	if strings.Contains(component, "..") {
		return fmt.Errorf("path component contains traversal pattern")
	}
	return nil
}
