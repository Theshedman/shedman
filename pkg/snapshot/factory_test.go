package snapshot

import (
	"testing"

	"github.com/theshedman/shedman/internal/config"
)

// Since we cannot easily mock util.GetRootFSType in this unit test without refactoring
// util to use an interface or global var, we will rely on integration tests or
//
// However, we CAN test the configuration override logic.

func TestFactory_GetManager_Override(t *testing.T) {
	cfg := config.Default()
	cfg.Snapshot.Backend = "rsync" // Force rsync

	f := NewFactory(cfg)
	mgr, err := f.GetManager()
	if err != nil {
		t.Fatalf("Failed to get manager: %v", err)
	}

	if mgr.GetBackendName() != "rsync" {
		t.Errorf("Expected rsync backend, got %s", mgr.GetBackendName())
	}
}

func TestFactory_GetManager_Auto(t *testing.T) {
	cfg := config.Default()
	cfg.Snapshot.Backend = "auto"

	f := NewFactory(cfg)
	mgr, err := f.GetManager()
	if err != nil {
		t.Fatalf("Failed to get manager: %v", err)
	}

	// We can't deterministicly know what detection will return on the build server
	// (it likely won't have snapper or timeshift), so it should fallback to "rsync".
	// Unless timeshift IS installed on this environment.
	// But "rsync" is the safe default if nothing satisfies.
	name := mgr.GetBackendName()
	if name == "" {
		t.Error("Backend name should not be empty")
	}
	t.Logf("Detected backend: %s", name)
}
