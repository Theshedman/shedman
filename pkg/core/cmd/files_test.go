package cmd

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

func TestRunFilesList(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.PkgFilesFunc = func(pkgName string) ([]string, error) {
		if pkgName == "vim" {
			return []string{"/usr/bin/vim", "/usr/share/vim/vimrc"}, nil
		}
		return nil, core.ErrPackageNotFound
	}

	// Helper to capture verify files listing (via engine interface directly for now)
	files, err := eng.GetOfficialBackend().GetPackageFiles("vim")
	if err != nil {
		t.Errorf("GetPackageFiles(vim) error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	// Test error case
	_, err = eng.GetOfficialBackend().GetPackageFiles("emacs")
	if err == nil {
		t.Error("Expected error for non-existent package")
	}
}

func TestRunFilesSearch(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.SearchFilesFunc = func(query string) ([]string, error) {
		if query == "vimrc" {
			return []string{"core/vim /usr/share/vim/vimrc", "extra/neovim /etc/xdg/nvim/sysinit.vim"}, nil
		}
		return []string{}, nil
	}

	files, err := eng.SearchFiles("vimrc")
	if err != nil {
		t.Errorf("SearchFiles(vimrc) error = %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}

	files, err = eng.SearchFiles("nothing")
	if err != nil {
		t.Errorf("SearchFiles(nothing) error = %v", err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, got %d", len(files))
	}
}
