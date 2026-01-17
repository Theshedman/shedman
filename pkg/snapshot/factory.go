package snapshot

import (
	"fmt"

	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/util"
	"github.com/theshedman/shedman/pkg/executor"
)

// Factory creates snapshot managers based on config and system state
type Factory struct {
	cfg  *config.Config
	exec executor.Executor
}

// NewFactory creates a new snapshot factory
func NewFactory(cfg *config.Config) *Factory {
	return &Factory{
		cfg:  cfg,
		exec: &executor.RealExecutor{},
	}

}

// GetScheduler returns a configured scheduler
func (f *Factory) GetScheduler() (Scheduler, error) {
	return NewSystemdScheduler(f.exec), nil
}

// GetKeyManager returns a configured key manager
func (f *Factory) GetKeyManager() (KeyManager, error) {
	return NewGPGKeyManager(f.exec), nil
}

// NewFactoryWithExecutor creates a factory with custom executor for testing
func NewFactoryWithExecutor(cfg *config.Config, executor executor.Executor) *Factory {
	return &Factory{
		cfg:  cfg,
		exec: executor,
	}

}

// GetManager returns the appropriate SnapshotManager
func (f *Factory) GetManager() (Manager, error) {
	if f.cfg.Snapshot.Backend != "" && f.cfg.Snapshot.Backend != "auto" {
		return f.createBackend(f.cfg.Snapshot.Backend)
	}

	backend := f.detectBackend()
	return f.createBackend(backend)
}

func (f *Factory) detectBackend() string {
	fsType := util.GetRootFSType()

	switch fsType {
	case "btrfs":
		if util.IsCommandAvailable("snapper") {
			return "snapper"
		}
		return "timeshift"

	case "zfs":
		return "zfs"

	default:
		if util.IsCommandAvailable("timeshift") {
			return "timeshift"
		}
		return "rsync"
	}
}

func (f *Factory) createBackend(name string) (Manager, error) {
	switch name {
	case "snapper":
		return NewSnapperBackend(f.cfg, f.exec), nil
	case "timeshift":
		return NewTimeshiftBackend(f.cfg, f.exec), nil
	case "rsync":
		return NewRsyncBackend(f.cfg, f.exec), nil
	default:
		return nil, fmt.Errorf("unknown snapshot backend: %s", name)
	}
}
