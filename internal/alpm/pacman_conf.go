package alpm

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/theshedman/shedman/internal/util"
)

// PacmanConf represents parsed pacman.conf configuration
type PacmanConf struct {
	RootDir      string
	DBPath       string
	CacheDir     string
	LogFile      string
	GPGDir       string
	HookDir      string
	HoldPkg      []string
	IgnorePkg    []string
	IgnoreGroup  []string
	Architecture string
	SigLevel     string
	Repositories []RepoConfig
}

// RepoConfig represents a repository configuration
type RepoConfig struct {
	Name     string
	Servers  []string
	SigLevel string
	Include  string
}

// DefaultPacmanConfPath is the default path to pacman.conf
const DefaultPacmanConfPath = "/etc/pacman.conf"

// ParsePacmanConf parses a pacman.conf file
func ParsePacmanConf(path string) (*PacmanConf, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	conf := &PacmanConf{
		RootDir:      "/",
		DBPath:       "/var/lib/pacman",
		CacheDir:     "/var/cache/pacman/pkg",
		LogFile:      "/var/log/pacman.log",
		GPGDir:       "/etc/pacman.d/gnupg",
		HookDir:      "/etc/pacman.d/hooks",
		Architecture: "auto",
		SigLevel:     "Required DatabaseOptional",
	}

	var currentRepo *RepoConfig
	scanner := bufio.NewScanner(file)
	sectionRe := regexp.MustCompile(`^\[([^\]]+)\]`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if match := sectionRe.FindStringSubmatch(line); match != nil {
			section := match[1]
			if section == "options" {
				currentRepo = nil
			} else {
				currentRepo = &RepoConfig{Name: section}
				conf.Repositories = append(conf.Repositories, *currentRepo)
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		var value string
		if len(parts) > 1 {
			value = strings.TrimSpace(parts[1])
		}

		if currentRepo == nil {
			switch key {
			case "RootDir":
				conf.RootDir = value
			case "DBPath":
				conf.DBPath = value
			case "CacheDir":
				conf.CacheDir = value
			case "LogFile":
				conf.LogFile = value
			case "GPGDir":
				conf.GPGDir = value
			case "HookDir":
				conf.HookDir = value
			case "HoldPkg":
				conf.HoldPkg = parseSpaceSeparated(value)
			case "IgnorePkg":
				conf.IgnorePkg = parseSpaceSeparated(value)
			case "IgnoreGroup":
				conf.IgnoreGroup = parseSpaceSeparated(value)
			case "Architecture":
				conf.Architecture = value
			case "SigLevel":
				conf.SigLevel = value
			}
		} else {
			idx := len(conf.Repositories) - 1
			switch key {
			case "Server":
				conf.Repositories[idx].Servers = append(conf.Repositories[idx].Servers, value)
			case "SigLevel":
				conf.Repositories[idx].SigLevel = value
			case "Include":
				conf.Repositories[idx].Include = value
				servers := parseMirrorlist(value, conf.Repositories[idx].Name)
				conf.Repositories[idx].Servers = append(conf.Repositories[idx].Servers, servers...)
			}
		}
	}

	return conf, scanner.Err()
}

// parseSpaceSeparated splits a space-separated value into a slice
func parseSpaceSeparated(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Fields(value)
}

// parseMirrorlist parses a mirrorlist file and returns server URLs
func parseMirrorlist(path, repoName string) []string {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Server") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				server := strings.TrimSpace(parts[1])
				server = strings.ReplaceAll(server, "$repo", repoName)
				servers = append(servers, server)
			}
		}
	}

	return servers
}

// DefaultPacmanConf returns a default configuration
func DefaultPacmanConf() *PacmanConf {
	return &PacmanConf{
		RootDir:      "/",
		DBPath:       "/var/lib/pacman",
		CacheDir:     "/var/cache/pacman/pkg",
		LogFile:      "/var/log/pacman.log",
		GPGDir:       "/etc/pacman.d/gnupg",
		HookDir:      "/etc/pacman.d/hooks",
		Architecture: "auto",
		SigLevel:     "Required DatabaseOptional",
		Repositories: []RepoConfig{
			{Name: "core", Include: "/etc/pacman.d/mirrorlist"},
			{Name: "extra", Include: "/etc/pacman.d/mirrorlist"},
			{Name: "multilib", Include: "/etc/pacman.d/mirrorlist"},
		},
	}
}

// GetArchitecture returns the resolved architecture
func (c *PacmanConf) GetArchitecture() string {
	if c.Architecture == "auto" {
		arch, err := os.ReadFile("/proc/sys/kernel/arch")
		if err == nil {
			return strings.TrimSpace(string(arch))
		}
		return "x86_64"
	}
	return c.Architecture
}

// ExpandVariables expands $repo and $arch in a URL
func (c *PacmanConf) ExpandVariables(url, repoName string) string {
	url = strings.ReplaceAll(url, "$repo", repoName)
	url = strings.ReplaceAll(url, "$arch", c.GetArchitecture())
	return url
}

// GetMirrorsForRepo returns all mirrors for a repository
func (c *PacmanConf) GetMirrorsForRepo(repoName string) []string {
	for _, repo := range c.Repositories {
		if repo.Name == repoName {
			var expanded []string
			for _, server := range repo.Servers {
				expanded = append(expanded, c.ExpandVariables(server, repoName))
			}
			return expanded
		}
	}
	return nil
}

// GetLogPath returns the log file path, ensuring parent directory exists
func (c *PacmanConf) GetLogPath() string {
	dir := filepath.Dir(c.LogFile)
	os.MkdirAll(dir, util.DirPermissions)
	return c.LogFile
}
