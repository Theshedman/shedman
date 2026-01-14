package cmd

import (
	"bytes"
	"strings"
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

func TestRunFiles(t *testing.T) {
	mock := &MockBackend{}
	eng := core.NewEngineWithBackend(mock)

	mock.PkgFilesFunc = func(pkgName string) ([]string, error) {
		if pkgName == "vim" {
			return []string{"/usr/bin/vim", "/usr/share/vim/vimrc"}, nil
		}
		return nil, core.ErrPackageNotFound
	}
	mock.SearchFilesFunc = func(query string) ([]string, error) {
		if query == "vimrc" {
			return []string{"core/vim /usr/share/vim/vimrc"}, nil
		}
		return []string{}, nil
	}

	var buf bytes.Buffer

	// Test List Files
	if err := RunFiles(eng, &buf, "vim", false); err != nil {
		t.Fatalf("RunFiles list failed: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "/usr/bin/vim") {
		t.Errorf("List output missing file. Got: %s", out)
	}

	// Test Search Files
	buf.Reset()
	if err := RunFiles(eng, &buf, "vimrc", true); err != nil {
		t.Fatalf("RunFiles search failed: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, "core/vim") {
		t.Errorf("Search output missing result. Got: %s", out)
	}

	// Test Missing Package (List)
	buf.Reset()
	if err := RunFiles(eng, &buf, "missing", false); err == nil {
		t.Error("Expected error for missing package")
	}
}
