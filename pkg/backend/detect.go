package backend

import (
	"bufio"
	"os"
	"os/exec"
	"strings"
)

// Arch-based distributions that support pacman and AUR
var archDistros = []string{
	"arch",
	"manjaro",
	"endeavouros",
	"artix",
	"cachyos",
	"shedos",
	"garuda",
	"arcolinux",
	"archcraft",
}

// DistroInfo contains detected distribution information
type DistroInfo struct {
	ID       string   // Primary ID from os-release
	IDLike   []string // Parent distros
	Name     string   // Pretty name
	Version  string   // Version string
	Family   string   // Detected family (arch, debian, fedora, suse)
	IsShedOS bool     // Is this ShedOS specifically
}

// DetectDistro reads /etc/os-release and returns distro info
func DetectDistro() DistroInfo {
	return detectDistroFromFile("/etc/os-release")
}

// detectDistroFromFile reads a specific os-release file (for testing)
func detectDistroFromFile(path string) DistroInfo {
	info := DistroInfo{}

	file, err := os.Open(path)
	if err != nil {
		return info
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "="); idx > 0 {
			key := line[:idx]
			value := strings.Trim(line[idx+1:], "\"")

			switch key {
			case "ID":
				info.ID = strings.ToLower(value)
			case "ID_LIKE":
				for _, v := range strings.Fields(value) {
					info.IDLike = append(info.IDLike, strings.ToLower(v))
				}
			case "NAME":
				info.Name = value
			case "VERSION", "VERSION_ID":
				if info.Version == "" {
					info.Version = value
				}
			}
		}
	}

	// Determine family
	info.Family = determineFamily(info.ID, info.IDLike)
	info.IsShedOS = info.ID == "shedos"

	return info
}

// determineFamily determines the distro family
func determineFamily(id string, idLike []string) string {
	// Check direct ID first
	if isArchID(id) {
		return "arch"
	}
	if isDebianID(id) {
		return "debian"
	}
	if isFedoraID(id) {
		return "fedora"
	}
	if isSuseID(id) {
		return "suse"
	}

	// Check ID_LIKE
	for _, parent := range idLike {
		if isArchID(parent) {
			return "arch"
		}
		if isDebianID(parent) {
			return "debian"
		}
		if isFedoraID(parent) {
			return "fedora"
		}
		if isSuseID(parent) {
			return "suse"
		}
	}

	return "unknown"
}

func isArchID(id string) bool {
	for _, d := range archDistros {
		if id == d {
			return true
		}
	}
	return false
}

func isDebianID(id string) bool {
	debianDistros := []string{"debian", "ubuntu", "mint", "pop", "elementary", "zorin", "kali", "raspbian"}
	for _, d := range debianDistros {
		if id == d {
			return true
		}
	}
	return false
}

func isFedoraID(id string) bool {
	fedoraDistros := []string{"fedora", "centos", "rhel", "rocky", "alma", "nobara"}
	for _, d := range fedoraDistros {
		if id == d {
			return true
		}
	}
	return false
}

func isSuseID(id string) bool {
	suseDistros := []string{"opensuse", "suse", "opensuse-leap", "opensuse-tumbleweed"}
	for _, d := range suseDistros {
		if id == d {
			return true
		}
	}
	return false
}

// IsArchBased returns true if the system is Arch-based
func IsArchBased() bool {
	info := DetectDistro()
	return info.Family == "arch"
}

// IsAURAvailable returns true if AUR is available on this system
// AUR is only available on Arch-based distributions
func IsAURAvailable() bool {
	return IsArchBased()
}

// DetectBackend auto-detects and returns the appropriate OfficialBackend
func DetectBackend() (OfficialBackend, error) {
	return DetectBackendWithConfig(nil)
}

// hasBinary checks if a binary is available in PATH
func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// GetBackendName returns the name of the detected backend
func GetBackendName() string {
	info := DetectDistro()

	switch info.Family {
	case "arch":
		return "pacman"
	case "debian":
		return "apt"
	case "fedora":
		return "dnf"
	case "suse":
		return "zypper"
	default:
		return "unknown"
	}
}
