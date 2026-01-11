package resolver_test

import (
	"testing"

	"github.com/theshedman/shedman/pkg/core/pkgdb"
	"github.com/theshedman/shedman/pkg/core/resolver"
)

func TestResolveRequest_ParseSimpleName(t *testing.T) {
	req := resolver.ParseRequest("neovim")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "" {
		t.Errorf("Expected empty Version, got '%s'", req.Version)
	}
	if req.Source != "" {
		t.Errorf("Expected empty Source, got '%s'", req.Source)
	}
}

func TestResolveRequest_ParseVersionConstraint(t *testing.T) {
	req := resolver.ParseRequest("neovim@0.10.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "0.10.0" {
		t.Errorf("Expected Version '0.10.0', got '%s'", req.Version)
	}
	if req.Operator != "=" {
		t.Errorf("Expected Operator '=', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseVersionGTE(t *testing.T) {
	req := resolver.ParseRequest("neovim>=0.9.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "0.9.0" {
		t.Errorf("Expected Version '0.9.0', got '%s'", req.Version)
	}
	if req.Operator != ">=" {
		t.Errorf("Expected Operator '>=', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseVersionLTE(t *testing.T) {
	req := resolver.ParseRequest("neovim<=1.0.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "1.0.0" {
		t.Errorf("Expected Version '1.0.0', got '%s'", req.Version)
	}
	if req.Operator != "<=" {
		t.Errorf("Expected Operator '<=', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseVersionGT(t *testing.T) {
	req := resolver.ParseRequest("neovim>0.8.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "0.8.0" {
		t.Errorf("Expected Version '0.8.0', got '%s'", req.Version)
	}
	if req.Operator != ">" {
		t.Errorf("Expected Operator '>', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseVersionLT(t *testing.T) {
	req := resolver.ParseRequest("neovim<2.0.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "2.0.0" {
		t.Errorf("Expected Version '2.0.0', got '%s'", req.Version)
	}
	if req.Operator != "<" {
		t.Errorf("Expected Operator '<', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseVersionExact(t *testing.T) {
	req := resolver.ParseRequest("neovim=0.10.0")

	if req.Name != "neovim" {
		t.Errorf("Expected Name 'neovim', got '%s'", req.Name)
	}
	if req.Version != "0.10.0" {
		t.Errorf("Expected Version '0.10.0', got '%s'", req.Version)
	}
	if req.Operator != "=" {
		t.Errorf("Expected Operator '=', got '%s'", req.Operator)
	}
}

func TestResolveRequest_ParseGroup(t *testing.T) {
	req := resolver.ParseRequest("@dev")

	if req.Name != "@dev" {
		t.Errorf("Expected Name '@dev', got '%s'", req.Name)
	}
	if !req.IsGroup {
		t.Error("Expected IsGroup to be true")
	}
}

func TestResolver_Resolve_SinglePackage(t *testing.T) {
	// Mock database
	db := &mockDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {Name: "neovim", Version: "0.10.0", Source: pkgdb.SourceOfficial},
		},
	}

	r := resolver.New(db)
	result, err := r.Resolve([]string{"neovim"})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	if len(result.ToInstall) != 1 {
		t.Errorf("Expected 1 package to install, got %d", len(result.ToInstall))
	}

	if result.ToInstall[0].Name != "neovim" {
		t.Errorf("Expected 'neovim', got '%s'", result.ToInstall[0].Name)
	}
}

func TestResolver_Resolve_WithDependency(t *testing.T) {
	db := &mockDB{
		packages: map[string]pkgdb.PackageInfo{
			"neovim": {Name: "neovim", Version: "0.10.0", Source: pkgdb.SourceOfficial, Depends: []string{"luajit"}},
			"luajit": {Name: "luajit", Version: "2.1", Source: pkgdb.SourceOfficial},
		},
	}

	r := resolver.New(db)
	result, err := r.Resolve([]string{"neovim"})

	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}

	// Should include both neovim and its dependency luajit
	if len(result.ToInstall) != 2 {
		t.Errorf("Expected 2 packages (neovim + luajit), got %d", len(result.ToInstall))
	}
}

// Mock database for testing
type mockDB struct {
	packages map[string]pkgdb.PackageInfo
}

func (m *mockDB) Search(query string) ([]pkgdb.PackageInfo, error) {
	var results []pkgdb.PackageInfo
	for _, p := range m.packages {
		results = append(results, p)
	}
	return results, nil
}

func (m *mockDB) GetInfo(name string) (*pkgdb.PackageInfo, error) {
	if p, ok := m.packages[name]; ok {
		return &p, nil
	}
	return nil, nil
}
