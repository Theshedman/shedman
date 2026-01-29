package disk

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/theshedman/shedman/pkg/executor"
)

// DeviceInfo represents a block device
type DeviceInfo struct {
	Name       string      `json:"name"`
	MountPoint string      `json:"mountpoint"`
	FSType     string      `json:"fstype"`
	Size       json.Number `json:"size"`
}

// BlockDevice represents the lsblk output structure
type blockDeviceOutput struct {
	BlockDevices []DeviceInfo `json:"blockdevices"`
}

// Manager handles disk operations
type Manager struct {
	exec executor.Executor
}

// NewManager creates a new Disk Manager
func NewManager(exec executor.Executor) *Manager {
	return &Manager{exec: exec}
}

// GetDeviceInfo returns info about a specific device (e.g. /dev/sdb1)
func (m *Manager) GetDeviceInfo(devicePath string) (*DeviceInfo, error) {
	// lsblk -J -o NAME,MOUNTPOINT,FSTYPE,SIZE /dev/sdb1
	out, err := m.exec.Command("lsblk", "-J", "-o", "NAME,MOUNTPOINT,FSTYPE,SIZE", devicePath).Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk failed: %w", err)
	}

	var output blockDeviceOutput
	if err := json.Unmarshal(out, &output); err != nil {
		return nil, fmt.Errorf("failed to parse lsblk output: %w", err)
	}

	if len(output.BlockDevices) == 0 {
		// Handle empty list case
		return nil, fmt.Errorf("device not found or not a block device")
	}

	return &output.BlockDevices[0], nil
}

// Mount mounts a device to a target path. returns cleanup function.
// targetPath matches /tmp/shedman-mount-* pattern if empty.
func (m *Manager) Mount(devicePath string) (string, func(), error) {
	info, err := m.GetDeviceInfo(devicePath)
	if err != nil {
		return "", nil, err
	}

	// Use existing mount if available
	if info.MountPoint != "" {
		return info.MountPoint, func() {}, nil // No cleanup needed
	}

	// Create temp mount point
	mountPoint, err := os.MkdirTemp("", "shedman-mount-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp mount point: %w", err)
	}

	// Mount device
	err = m.exec.Command("sudo", "mount", devicePath, mountPoint).Run()

	// If direct mount failed, we return error.
	if err != nil {
		_ = os.Remove(mountPoint)
		return "", nil, fmt.Errorf("failed to mount %s: %w", devicePath, err)
	}

	cleanup := func() {
		// Unmount
		_ = m.exec.Command("sudo", "umount", mountPoint).Run()
		_ = os.Remove(mountPoint)
	}

	return mountPoint, cleanup, nil
}

// CheckSafeguards checks if device is safe to use
func (m *Manager) CheckSafeguards(devicePath string) error {
	info, err := m.GetDeviceInfo(devicePath)
	if err != nil {
		return err
	}
	if info.FSType == "" {
		return fmt.Errorf("device %s has no filesystem type (unformatted?)", devicePath)
	}
	if info.FSType == "swap" {
		return fmt.Errorf("device %s is swap", devicePath)
	}
	return nil
}
