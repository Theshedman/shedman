package core

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func (e *Engine) runHook(path, action string, pkgs []string) error {
	if e == nil || e.config == nil || path == "" {
		return nil
	}

	cmd := exec.Command(path)
	cmd.Env = append(os.Environ(), formatHookEnv(action, pkgs)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook %s failed: %w", path, err)
	}
	return nil
}

func formatHookEnv(action string, pkgs []string) []string {
	env := []string{
		"SHEDMAN_ACTION=" + action,
	}
	if len(pkgs) > 0 {
		env = append(env, "SHEDMAN_PACKAGES="+strings.Join(pkgs, " "))
	}
	return env
}
