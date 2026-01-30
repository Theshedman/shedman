package security

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core"
)

type mockBackend struct {
	audit []string
}

func (m *mockBackend) Name() string      { return "mock" }
func (m *mockBackend) Sync() error       { return nil }
func (m *mockBackend) IsAvailable() bool { return true }
func (m *mockBackend) Install(pkgs []string, opts core.InstallOptions) error {
	return nil
}
func (m *mockBackend) Remove(pkgs []string, opts core.RemoveOptions) error { return nil }
func (m *mockBackend) IsInstalled(pkgName string) bool                     { return false }
func (m *mockBackend) Search(query string) ([]core.PackageInfo, error)     { return nil, nil }
func (m *mockBackend) Info(pkgName string) (*core.PackageInfo, error)      { return nil, nil }
func (m *mockBackend) GetInstalledPackages() ([]core.PackageInfo, error)   { return nil, nil }
func (m *mockBackend) Upgrade(pkgs []string, opts core.UpgradeOptions) error {
	return nil
}
func (m *mockBackend) InstallLocal(path string, opts core.InstallOptions) error {
	return nil
}
func (m *mockBackend) GetPackageFiles(pkgName string) ([]string, error) { return nil, nil }
func (m *mockBackend) GetFileOwner(path string) (string, error)         { return "", nil }
func (m *mockBackend) SearchFiles(query string) ([]string, error)       { return nil, nil }
func (m *mockBackend) ListExplicitPackages() ([]string, error)          { return nil, nil }
func (m *mockBackend) Audit() ([]string, error)                         { return m.audit, nil }
func (m *mockBackend) Diff() ([]core.PackageDiff, error)                { return nil, nil }

func TestNew(t *testing.T) {
	// core can be nil for this stub test as it's not used yet
	var engine *core.Engine
	s := New(engine)
	if s == nil {
		t.Fatal("New returned nil")
	}
	if s.core != engine {
		t.Error("Engine not set correctly")
	}
}

func TestScanner_Check(t *testing.T) {
	mock := &mockBackend{
		audit: []string{
			"openssl 1.1.1k-1 (CVE-2023-1234) HIGH: tls issue",
			"bash 5.1 (CVE-2020-1111) (CVE-2020-2222) medium",
		},
	}
	engine := core.NewEngineWithBackend(mock)
	s := New(engine)

	vulns, err := s.Check()
	if err != nil {
		t.Errorf("Check returned error: %v", err)
	}
	if len(vulns) != 3 {
		t.Fatalf("Expected 3 vulnerabilities, got %d", len(vulns))
	}

	if vulns[0].Package != "openssl" || vulns[0].CVE != "CVE-2023-1234" || vulns[0].Severity != "high" {
		t.Errorf("Unexpected parsed vulnerability: %+v", vulns[0])
	}
	if vulns[1].Package != "bash" || vulns[1].Severity != "medium" {
		t.Errorf("Unexpected parsed vulnerability: %+v", vulns[1])
	}
}

func TestScanner_Check_NoEngine(t *testing.T) {
	s := New(nil)

	if _, err := s.Check(); err == nil {
		t.Fatal("Expected error for nil engine")
	}
}
