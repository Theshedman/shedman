package pacman

import (
	"errors"
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

// MockExecutor implements CommandExecutor for testing
type MockExecutor struct {
	RunFunc        func(name string, args ...string) error
	SilentRunFunc  func(name string, args ...string) error
	OutputFunc     func(name string, args ...string) ([]byte, error)
	RunCalls       [][]string
	SilentRunCalls [][]string
	OutputCalls    [][]string
}

func (m *MockExecutor) Run(name string, args ...string) error {
	call := append([]string{name}, args...)
	m.RunCalls = append(m.RunCalls, call)
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil
}

func (m *MockExecutor) SilentRun(name string, args ...string) error {
	call := append([]string{name}, args...)
	m.SilentRunCalls = append(m.SilentRunCalls, call)
	if m.SilentRunFunc != nil {
		return m.SilentRunFunc(name, args...)
	}
	return nil
}

func (m *MockExecutor) Output(name string, args ...string) ([]byte, error) {
	call := append([]string{name}, args...)
	m.OutputCalls = append(m.OutputCalls, call)
	if m.OutputFunc != nil {
		return m.OutputFunc(name, args...)
	}
	return []byte{}, nil
}

func TestBackend_Name(t *testing.T) {
	b := NewWithExecutor(&MockExecutor{})
	if b.Name() != "pacman" {
		t.Errorf("Expected name 'pacman', got '%s'", b.Name())
	}
}

func TestBackend_Install_BuildsCorrectCommand(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	opts := core.InstallOptions{
		Needed:    true,
		NoConfirm: true,
		AsDeps:    true,
	}

	err := b.Install([]string{"vim", "git"}, opts)
	if err != nil {
		t.Fatalf("Install failed: %v", err)
	}

	if len(mock.RunCalls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.RunCalls))
	}

	call := mock.RunCalls[0]
	// Should be: sudo pacman -S --needed --asdeps --noconfirm vim git
	expected := []string{"sudo", "pacman", "-S", "--needed", "--asdeps", "--noconfirm", "vim", "git"}
	if len(call) != len(expected) {
		t.Errorf("Expected %d args, got %d: %v", len(expected), len(call), call)
	}
}

func TestBackend_Install_EmptyPackages(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	err := b.Install([]string{}, core.InstallOptions{})
	if err != nil {
		t.Fatalf("Install with empty packages should not error: %v", err)
	}

	if len(mock.RunCalls) != 0 {
		t.Errorf("Should not call executor for empty packages")
	}
}

func TestBackend_Remove_BuildsCorrectCommand(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	opts := core.RemoveOptions{
		Cascade:   true,
		Recursive: true,
		NoConfirm: true,
	}

	err := b.Remove([]string{"vim"}, opts)
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	call := mock.RunCalls[0]
	// Should contain: sudo pacman -R -c -s --noconfirm vim
	found := map[string]bool{}
	for _, arg := range call {
		found[arg] = true
	}

	if !found["-c"] {
		t.Error("Expected -c flag for cascade")
	}
	if !found["-s"] {
		t.Error("Expected -s flag for recursive")
	}
	if !found["--noconfirm"] {
		t.Error("Expected --noconfirm flag")
	}
}

func TestBackend_Sync_BuildsCorrectCommand(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	err := b.Sync()
	if err != nil {
		t.Fatalf("Sync failed: %v", err)
	}

	if len(mock.RunCalls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mock.RunCalls))
	}

	call := mock.RunCalls[0]
	// Should be: sudo pacman -Sy
	if call[0] != "sudo" || call[1] != "pacman" || call[2] != "-Sy" {
		t.Errorf("Unexpected command: %v", call)
	}
}

func TestBackend_Search_ParsesOutput(t *testing.T) {
	mock := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(`extra/vim 9.0-1
    Vi Improved - a highly configurable text editor
extra/git 2.42.0-1
    Fast distributed version control system
`), nil
		},
	}
	b := NewWithExecutor(mock)

	results, err := b.Search("vim")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	if results[0].Name != "vim" {
		t.Errorf("Expected name 'vim', got '%s'", results[0].Name)
	}
	if results[0].Version != "9.0-1" {
		t.Errorf("Expected version '9.0-1', got '%s'", results[0].Version)
	}
}

func TestBackend_Info_ParsesOutput(t *testing.T) {
	mock := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(`Name            : vim
Version         : 9.0-1
Description     : Vi Improved - a highly configurable text editor
Depends On      : glibc ncurses
`), nil
		},
	}
	b := NewWithExecutor(mock)

	info, err := b.Info("vim")
	if err != nil {
		t.Fatalf("Info failed: %v", err)
	}

	if info.Name != "vim" {
		t.Errorf("Expected name 'vim', got '%s'", info.Name)
	}
	if info.Version != "9.0-1" {
		t.Errorf("Expected version '9.0-1', got '%s'", info.Version)
	}
	if len(info.Depends) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(info.Depends))
	}
}

func TestBackend_Info_NotFound(t *testing.T) {
	mock := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("package not found")
		},
	}
	b := NewWithExecutor(mock)

	_, err := b.Info("nonexistent")
	if err != core.ErrPackageNotFound {
		t.Errorf("Expected ErrPackageNotFound, got %v", err)
	}
}

func TestBackend_GetInstalledPackages_ParsesOutput(t *testing.T) {
	mock := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(`vim 9.0-1
git 2.42.0-1
`), nil
		},
	}
	b := NewWithExecutor(mock)

	pkgs, err := b.GetInstalledPackages()
	if err != nil {
		t.Fatalf("GetInstalledPackages failed: %v", err)
	}

	if len(pkgs) != 2 {
		t.Fatalf("Expected 2 packages, got %d", len(pkgs))
	}

	if pkgs[0].Name != "vim" || pkgs[0].Version != "9.0-1" {
		t.Errorf("Unexpected first package: %+v", pkgs[0])
	}
}

func TestBackend_GetPackageFiles_ParsesOutput(t *testing.T) {
	mock := &MockExecutor{
		OutputFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(`vim /usr/bin/vim
vim /usr/share/vim/
`), nil
		},
	}
	b := NewWithExecutor(mock)

	files, err := b.GetPackageFiles("vim")
	if err != nil {
		t.Fatalf("GetPackageFiles failed: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(files))
	}

	if files[0] != "/usr/bin/vim" {
		t.Errorf("Unexpected first file: %s", files[0])
	}
}

func TestBackend_IsInstalled(t *testing.T) {
	mock := &MockExecutor{
		SilentRunFunc: func(name string, args ...string) error {
			if len(args) > 1 && args[1] == "vim" {
				return nil
			}
			return errors.New("not installed")
		},
	}
	b := NewWithExecutor(mock)

	if !b.IsInstalled("vim") {
		t.Error("Expected vim to be installed")
	}
}

func TestBackend_InstallLocal_BuildsCorrectCommand(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	opts := core.InstallOptions{
		NoConfirm: true,
		AsDeps:    true,
	}

	err := b.InstallLocal("/path/to/pkg.tar.zst", opts)
	if err != nil {
		t.Fatalf("InstallLocal failed: %v", err)
	}

	call := mock.RunCalls[0]
	// Should be: sudo pacman -U --asdeps --noconfirm /path/to/pkg.tar.zst
	found := map[string]bool{}
	for _, arg := range call {
		found[arg] = true
	}

	if !found["-U"] {
		t.Error("Expected -U flag")
	}
	if !found["--asdeps"] {
		t.Error("Expected --asdeps flag")
	}
	if !found["/path/to/pkg.tar.zst"] {
		t.Error("Expected package path")
	}
}

func TestBackend_Upgrade_BuildsCorrectCommand(t *testing.T) {
	mock := &MockExecutor{}
	b := NewWithExecutor(mock)

	opts := core.UpgradeOptions{
		Refresh:   true,
		NoConfirm: true,
	}

	err := b.Upgrade([]string{}, opts)
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}

	call := mock.RunCalls[0]
	found := map[string]bool{}
	for _, arg := range call {
		found[arg] = true
	}

	if !found["-S"] {
		t.Error("Expected -S flag")
	}
	if !found["-y"] {
		t.Error("Expected -y flag for refresh")
	}
	if !found["-u"] {
		t.Error("Expected -u flag for upgrade")
	}
}
