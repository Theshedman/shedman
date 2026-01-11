package core

import (
	"errors"
	"os/exec"
)

// Common errors
var (
	ErrGitNotFound     = errors.New("git not found in PATH")
	ErrMakepkgNotFound = errors.New("makepkg not found in PATH")
	ErrBwrapNotFound   = errors.New("bwrap (bubblewrap) not found in PATH")
)

// CommandExists checks if a command exists in the PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Common file permissions
const (
	DirPermissions  = 0755
	FilePermissions = 0644
)
