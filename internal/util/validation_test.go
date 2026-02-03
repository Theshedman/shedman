package util

import (
	"strings"
	"testing"
)

func TestValidateUsername(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		username string
		wantErr  bool
	}{
		{"valid simple", "john", false},
		{"valid with underscore prefix", "_admin", false},
		{"valid with numbers", "user123", false},
		{"valid with hyphen", "john-doe", false},
		{"valid with underscore", "john_doe", false},
		{"valid root", "root", false},
		{"valid with uppercase", "John", false},
		{"valid mixed case", "JohnDoe", false},
		{"empty", "", true},
		{"starts with number", "1user", true},
		{"starts with hyphen", "-user", true},
		{"contains space", "john doe", true},
		{"contains semicolon", "john;rm -rf /", true},
		{"contains backtick", "john`whoami`", true},
		{"contains dollar", "john$HOME", true},
		{"contains slash", "john/doe", true},
		{"path traversal attempt", "../../../etc/passwd", true},
		{"command injection attempt", "x; rm -rf / #", true},
		{"too long", "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateUsername(tt.username)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUsername(%q) error = %v, wantErr %v", tt.username, err, tt.wantErr)
			}
		})
	}
}

func TestValidateZFSDatasetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		dataset string
		wantErr bool
	}{
		{"valid pool", "rpool", false},
		{"valid dataset", "rpool/ROOT/default", false},
		{"valid with snapshot", "rpool/ROOT@snapshot1", false},
		{"valid with hyphen", "rpool/my-data", false},
		{"valid with underscore", "rpool/my_data", false},
		{"valid with dot", "rpool/my.data", false},
		{"empty", "", true},
		{"starts with slash", "/rpool/data", true},
		{"ends with slash", "rpool/data/", true},
		{"double slash", "rpool//data", true},
		{"contains semicolon", "rpool; rm -rf /", true},
		{"contains backtick", "rpool`whoami`", true},
		{"contains dollar", "rpool$HOME", true},
		{"contains space", "rpool data", true},
		{"command injection attempt", "pool; reboot; echo", true},
		{"too long", strings.Repeat("a", 256), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateZFSDatasetName(tt.dataset)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateZFSDatasetName(%q) error = %v, wantErr %v", tt.dataset, err, tt.wantErr)
			}
		})
	}
}

func TestValidateHostname(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"valid hostname", "server.example.com", false},
		{"valid simple", "localhost", false},
		{"valid with numbers", "server1.example.com", false},
		{"valid IPv4", "192.168.1.1", false},
		{"valid IPv6", "::1", false},
		{"valid IPv6 full", "2001:db8::1", false},
		{"valid user@host", "user@server.example.com", false},
		{"empty", "", true},
		{"contains semicolon", "server; rm -rf /", true},
		{"contains backtick", "server`whoami`", true},
		{"contains pipe", "server | cat /etc/passwd", true},
		{"contains dollar", "server$HOME", true},
		{"contains ampersand", "server && reboot", true},
		{"command injection", "$(reboot).example.com", true},
		{"label too long", strings.Repeat("a", 64) + ".com", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateHostname(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateHostname(%q) error = %v, wantErr %v", tt.host, err, tt.wantErr)
			}
		})
	}
}

func TestValidateExecutablePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Only test cases that don't require actual file system access
		{"empty", "", true},
		{"contains semicolon", "/usr/bin/vim; rm -rf /", true},
		{"contains backtick", "/usr/bin/vim`whoami`", true},
		{"contains pipe", "/usr/bin/vim | cat", true},
		{"contains dollar", "/usr/bin/$EDITOR", true},
		{"contains ampersand", "/usr/bin/vim && reboot", true},
		{"command injection", "$(reboot)", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateExecutablePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateExecutablePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		component string
		wantErr   bool
	}{
		{"valid simple", "filename", false},
		{"valid with hyphen", "my-file", false},
		{"valid with underscore", "my_file", false},
		{"valid with dot", "my.file.txt", false},
		{"valid hidden file", ".bashrc", false},
		{"empty", "", true},
		{"forward slash", "path/to/file", true},
		{"backslash", "path\\to\\file", true},
		{"current dir", ".", true},
		{"parent dir", "..", true},
		{"traversal in name", "foo..bar", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := SanitizePath(tt.component)
			if (err != nil) != tt.wantErr {
				t.Errorf("SanitizePath(%q) error = %v, wantErr %v", tt.component, err, tt.wantErr)
			}
		})
	}
}
