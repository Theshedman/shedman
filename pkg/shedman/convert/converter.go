package convert

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// PackageConverter converts between package formats
type PackageConverter struct {
	cacheDir string
	executor Executor
}

// NewPackageConverter creates a new converter with default cache directory
func NewPackageConverter() *PackageConverter {
	return NewPackageConverterWithCacheDir("")
}

// NewPackageConverterWithCacheDir creates a new converter with custom cache directory
func NewPackageConverterWithCacheDir(cacheDir string) *PackageConverter {
	if cacheDir == "" {
		home, _ := os.UserHomeDir()
		cacheDir = filepath.Join(home, ".cache", "shedman", "convert")
	}
	return &PackageConverter{
		cacheDir: cacheDir,
		executor: DefaultExecutor,
	}
}

// SetExecutor sets a custom command executor (for testing)
func (c *PackageConverter) SetExecutor(exec Executor) {
	c.executor = exec
}

// SetCacheDir sets the cache directory (for testing)
func (c *PackageConverter) SetCacheDir(dir string) {
	c.cacheDir = dir
}

// PacmanPkgInfo represents .PKGINFO from a pacman package
type PacmanPkgInfo struct {
	PkgName    string
	PkgVer     string
	PkgDesc    string
	Depends    []string
	OptDepends []string
	Provides   []string
	Conflicts  []string
	Url        string
	Packager   string
	BuildDate  string
	Size       int64
	Arch       string
}

// ParsePkgInfo parses .PKGINFO file content
func ParsePkgInfo(content string) *PacmanPkgInfo {
	info := &PacmanPkgInfo{}

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " = ", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "pkgname":
			info.PkgName = value
		case "pkgver":
			info.PkgVer = value
		case "pkgdesc":
			info.PkgDesc = value
		case "depend":
			info.Depends = append(info.Depends, value)
		case "optdepend":
			info.OptDepends = append(info.OptDepends, value)
		case "provides":
			info.Provides = append(info.Provides, value)
		case "conflict":
			info.Conflicts = append(info.Conflicts, value)
		case "url":
			info.Url = value
		case "packager":
			info.Packager = value
		case "builddate":
			info.BuildDate = value
		case "size":
			fmt.Sscanf(value, "%d", &info.Size)
		case "arch":
			info.Arch = value
		}
	}

	return info
}

// ConvertPacmanToShed converts a .pkg.tar.zst/.pkg.tar.xz/.pkg.tar.gz to .shed format
func (c *PackageConverter) ConvertPacmanToShed(pacmanPkg string) (string, error) {
	// Create extraction directory
	pkgBase := filepath.Base(pacmanPkg)
	pkgBase = strings.TrimSuffix(pkgBase, ".pkg.tar.zst")
	pkgBase = strings.TrimSuffix(pkgBase, ".pkg.tar.xz")
	pkgBase = strings.TrimSuffix(pkgBase, ".pkg.tar.gz")

	extractDir := filepath.Join(c.cacheDir, "extract", pkgBase)
	shedDir := filepath.Join(c.cacheDir, "shed", pkgBase)

	// Clean up any existing directories
	os.RemoveAll(extractDir)
	os.RemoveAll(shedDir)

	// Create directories
	if err := os.MkdirAll(extractDir, DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create extract directory: %w", err)
	}
	if err := os.MkdirAll(shedDir, DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create shed directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(shedDir, "files"), DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create files directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(shedDir, "hooks"), DirPermissions); err != nil {
		return "", fmt.Errorf("failed to create hooks directory: %w", err)
	}

	// Extract pacman package using tar command (handles zstd)
	cmd := []string{"tar", "-xf", pacmanPkg, "-C", extractDir}
	if err := c.executor(cmd); err != nil {
		return "", fmt.Errorf("failed to extract pacman package: %w", err)
	}

	// Read .PKGINFO
	pkgInfoPath := filepath.Join(extractDir, ".PKGINFO")
	pkgInfoContent, err := os.ReadFile(pkgInfoPath)
	if err != nil {
		return "", fmt.Errorf("failed to read .PKGINFO: %w", err)
	}

	pkgInfo := ParsePkgInfo(string(pkgInfoContent))

	// Create manifest.toml
	manifest := ShedManifest{
		Name:        pkgInfo.PkgName,
		Version:     pkgInfo.PkgVer,
		Description: pkgInfo.PkgDesc,
		Depends:     pkgInfo.Depends,
		Provides:    pkgInfo.Provides,
		Conflicts:   pkgInfo.Conflicts,
	}

	manifestPath := filepath.Join(shedDir, "manifest.toml")
	manifestData, err := toml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, FilePermissions); err != nil {
		return "", fmt.Errorf("failed to write manifest: %w", err)
	}

	// Copy files (excluding .PKGINFO, .MTREE, .INSTALL, .BUILDINFO, .CHANGELOG)
	filesDir := filepath.Join(shedDir, "files")
	err = filepath.Walk(extractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(extractDir, path)

		// Skip metadata files
		if strings.HasPrefix(relPath, ".") {
			if !info.IsDir() {
				return nil
			}
			// Skip hidden directories
			if relPath == "." {
				return nil // Don't skip root
			}
			return filepath.SkipDir
		}

		destPath := filepath.Join(filesDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return copyFile(path, destPath, info.Mode())
	})
	if err != nil {
		return "", fmt.Errorf("failed to copy files: %w", err)
	}

	// Convert .INSTALL to hooks if present
	installPath := filepath.Join(extractDir, ".INSTALL")
	if _, err := os.Stat(installPath); err == nil {
		if err := c.convertInstallToHooks(installPath, filepath.Join(shedDir, "hooks")); err != nil {
			// Non-fatal, just log
			fmt.Printf("Warning: failed to convert .INSTALL to hooks: %v\n", err)
		}
	}

	// Create .shed archive
	shedFile := filepath.Join(c.cacheDir, pkgBase+".shed")
	cmd = []string{"tar", "-cf", shedFile, "-C", shedDir, "."}
	if err := c.executor(cmd); err != nil {
		return "", fmt.Errorf("failed to create .shed package: %w", err)
	}

	return shedFile, nil
}

// copyFile copies a file preserving permissions
func copyFile(src, dst string, mode os.FileMode) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), DirPermissions); err != nil {
		return err
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// convertInstallToHooks converts pacman .INSTALL script to .shed hooks
func (c *PackageConverter) convertInstallToHooks(installPath, hooksDir string) error {
	content, err := os.ReadFile(installPath)
	if err != nil {
		return err
	}

	script := string(content)

	// Extract pre_install function
	if strings.Contains(script, "pre_install") {
		preInstall := extractFunction(script, "pre_install")
		if preInstall != "" {
			hookPath := filepath.Join(hooksDir, "pre-install.sh")
			hookContent := "#!/bin/sh\n" + preInstall
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	// Extract post_install function
	if strings.Contains(script, "post_install") {
		postInstall := extractFunction(script, "post_install")
		if postInstall != "" {
			hookPath := filepath.Join(hooksDir, "post-install.sh")
			hookContent := "#!/bin/sh\n" + postInstall
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	// Extract pre_upgrade function
	if strings.Contains(script, "pre_upgrade") {
		preUpgrade := extractFunction(script, "pre_upgrade")
		if preUpgrade != "" {
			hookPath := filepath.Join(hooksDir, "pre-upgrade.sh")
			hookContent := "#!/bin/sh\n" + preUpgrade
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	// Extract post_upgrade function
	if strings.Contains(script, "post_upgrade") {
		postUpgrade := extractFunction(script, "post_upgrade")
		if postUpgrade != "" {
			hookPath := filepath.Join(hooksDir, "post-upgrade.sh")
			hookContent := "#!/bin/sh\n" + postUpgrade
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	// Extract pre_remove function
	if strings.Contains(script, "pre_remove") {
		preRemove := extractFunction(script, "pre_remove")
		if preRemove != "" {
			hookPath := filepath.Join(hooksDir, "pre-remove.sh")
			hookContent := "#!/bin/sh\n" + preRemove
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	// Extract post_remove function
	if strings.Contains(script, "post_remove") {
		postRemove := extractFunction(script, "post_remove")
		if postRemove != "" {
			hookPath := filepath.Join(hooksDir, "post-remove.sh")
			hookContent := "#!/bin/sh\n" + postRemove
			if err := os.WriteFile(hookPath, []byte(hookContent), ExecPermissions); err != nil {
				return err
			}
		}
	}

	return nil
}

// extractFunction extracts a shell function body from a script
func extractFunction(script, funcName string) string {
	// Simple extraction - find function and extract body
	// Format: funcName() { ... }
	patterns := []string{
		funcName + "()",
		funcName + " ()",
	}

	for _, pattern := range patterns {
		idx := strings.Index(script, pattern)
		if idx == -1 {
			continue
		}

		// Find opening brace
		braceIdx := strings.Index(script[idx:], "{")
		if braceIdx == -1 {
			continue
		}

		// Find matching closing brace
		start := idx + braceIdx + 1
		depth := 1
		end := start

		for i := start; i < len(script) && depth > 0; i++ {
			switch script[i] {
			case '{':
				depth++
			case '}':
				depth--
			}
			if depth == 0 {
				end = i
				break
			}
		}

		if end > start {
			return strings.TrimSpace(script[start:end])
		}
	}

	return ""
}

// DetectPackageFormat detects the format of a package file
func DetectPackageFormat(path string) string {
	ext := filepath.Ext(path)
	base := filepath.Base(path)

	switch {
	case ext == ".shed":
		return "shed"
	case strings.HasSuffix(base, ".pkg.tar.zst"):
		return "pacman-zst"
	case strings.HasSuffix(base, ".pkg.tar.xz"):
		return "pacman-xz"
	case strings.HasSuffix(base, ".pkg.tar.gz"):
		return "pacman-gz"
	case ext == ".deb":
		return "deb"
	case ext == ".rpm":
		return "rpm"
	default:
		return "unknown"
	}
}
