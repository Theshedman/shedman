package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

func TestRunHook(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		engine      *Engine
		path        string
		action      string
		pkgs        []string
		wantErr     bool
		skipOnCI    bool
		setupScript string
	}{
		{
			name:   "nil engine",
			engine: nil,
			path:   "/usr/bin/true",
			action: "install",
			pkgs:   []string{"foo"},
		},
		{
			name:   "nil config",
			engine: &Engine{config: nil},
			path:   "/usr/bin/true",
			action: "install",
			pkgs:   []string{"foo"},
		},
		{
			name:   "empty path",
			engine: &Engine{config: config.Default()},
			path:   "",
			action: "install",
			pkgs:   []string{"foo"},
		},
		{
			name:        "successful hook",
			engine:      &Engine{config: config.Default()},
			path:        "will_be_replaced",
			action:      "install",
			pkgs:        []string{"foo", "bar"},
			wantErr:     false,
			setupScript: "#!/bin/sh\nexit 0",
		},
		{
			name:        "failing hook",
			engine:      &Engine{config: config.Default()},
			path:        "will_be_replaced",
			action:      "install",
			pkgs:        []string{"foo"},
			wantErr:     true,
			setupScript: "#!/bin/sh\nexit 1",
		},
		{
			name:    "non-existent hook",
			engine:  &Engine{config: config.Default()},
			path:    "/nonexistent/hook/path",
			action:  "install",
			pkgs:    []string{"foo"},
			wantErr: true,
		},
		{
			name:        "hook receives environment variables",
			engine:      &Engine{config: config.Default()},
			path:        "will_be_replaced",
			action:      "upgrade",
			pkgs:        []string{"pkg1", "pkg2", "pkg3"},
			wantErr:     false,
			setupScript: "#!/bin/sh\n[ \"$SHEDMAN_ACTION\" = \"upgrade\" ] && [ \"$SHEDMAN_PACKAGES\" = \"pkg1 pkg2 pkg3\" ]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			path := tt.path

			// If we need to create a script, do it in a temp directory
			if tt.setupScript != "" {
				tmpDir := t.TempDir()
				scriptPath := filepath.Join(tmpDir, "hook.sh")
				if err := os.WriteFile(scriptPath, []byte(tt.setupScript), 0755); err != nil {
					t.Fatalf("Failed to create test script: %v", err)
				}
				path = scriptPath
			}

			err := tt.engine.runHook(path, tt.action, tt.pkgs)
			if (err != nil) != tt.wantErr {
				t.Errorf("runHook() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFormatHookEnv(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		action   string
		pkgs     []string
		wantVars map[string]string
	}{
		{
			name:   "action only",
			action: "install",
			pkgs:   nil,
			wantVars: map[string]string{
				"SHEDMAN_ACTION": "install",
			},
		},
		{
			name:   "action with packages",
			action: "remove",
			pkgs:   []string{"foo", "bar"},
			wantVars: map[string]string{
				"SHEDMAN_ACTION":   "remove",
				"SHEDMAN_PACKAGES": "foo bar",
			},
		},
		{
			name:   "empty packages slice",
			action: "upgrade",
			pkgs:   []string{},
			wantVars: map[string]string{
				"SHEDMAN_ACTION": "upgrade",
			},
		},
		{
			name:   "single package",
			action: "sync",
			pkgs:   []string{"single"},
			wantVars: map[string]string{
				"SHEDMAN_ACTION":   "sync",
				"SHEDMAN_PACKAGES": "single",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			env := formatHookEnv(tt.action, tt.pkgs)

			// Convert to map for easier checking
			envMap := make(map[string]string)
			for _, e := range env {
				parts := splitEnvVar(e)
				if len(parts) == 2 {
					envMap[parts[0]] = parts[1]
				}
			}

			for key, want := range tt.wantVars {
				got, ok := envMap[key]
				if !ok {
					t.Errorf("missing expected env var %s", key)
					continue
				}
				if got != want {
					t.Errorf("env var %s = %q, want %q", key, got, want)
				}
			}

			// Check for unexpected SHEDMAN_PACKAGES when not expected
			if _, hasPackages := tt.wantVars["SHEDMAN_PACKAGES"]; !hasPackages {
				if _, found := envMap["SHEDMAN_PACKAGES"]; found {
					t.Errorf("unexpected SHEDMAN_PACKAGES in env")
				}
			}
		})
	}
}

// splitEnvVar splits KEY=VALUE into [KEY, VALUE]
func splitEnvVar(s string) []string {
	for i, c := range s {
		if c == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func TestRunHookPermissionDenied(t *testing.T) {
	t.Parallel()

	// Skip if running as root (permissions won't be denied)
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "hook.sh")

	// Create a script without execute permissions
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0"), 0644); err != nil {
		t.Fatalf("Failed to create test script: %v", err)
	}

	engine := &Engine{config: config.Default()}
	err := engine.runHook(scriptPath, "install", []string{"foo"})

	if err == nil {
		t.Error("expected error for non-executable hook, got nil")
	}
}
