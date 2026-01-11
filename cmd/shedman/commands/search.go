package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/pkg/backend"
	"github.com/theshedman/shedman/pkg/backend/aur"
	"github.com/theshedman/shedman/pkg/backend/shedrepo"
	"github.com/theshedman/shedman/internal/cache"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/pkg/core/installer"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core/pkgdb"
)

var (
	searchOfficial  bool
	searchAUR       bool
	searchShedOS    bool
	searchInstalled bool
	searchJSON      bool
	searchLimit     int
)

// SearchResult holds a search result with source information
type SearchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Installed   bool   `json:"installed"`
}

var SEARCHCMD = &cobra.Command{
	Use:   "search <query>",
	Short: "Search for packages",
	Long: `Search for packages across multiple sources.

Sources searched:
  - Official repositories (via distro's native package manager)
  - AUR (Arch User Repository, if enabled and on Arch-based system)
  - ShedOS repository
  - Installed packages

Examples:
  shedman search neovim           # Search all sources
  shedman search neovim --official  # Official repos only
  shedman search neovim --aur     # AUR only
  shedman search neovim --shedos  # ShedOS only
  shedman search neovim --installed # Installed only
  shedman search neovim --json    # JSON output`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := strings.Join(args, " ")

		// Load configuration
		cfg, err := config.LoadDefault()
		if err != nil {
			output.Warning("Failed to load config: %v", err)
			cfg = config.Default()
		}

		// Determine which sources to search
		searchAll := !searchOfficial && !searchAUR && !searchShedOS && !searchInstalled

		var results []SearchResult
		var searchErrors []string

		// Create shared cache instance (reused across backends)
		cacheDir := cache.NewFileSystemCache()

		// Get official backend
		officialBackend, err := backend.DetectBackendWithConfig(&cfg.Backend)
		if err != nil {
			output.Warning("Official backend not available: %v", err)
			officialBackend = nil
		}

		// Search official repositories
		if searchAll || searchOfficial {
			if officialBackend != nil {
				pkgs, err := officialBackend.Search(query)
				if err != nil {
					searchErrors = append(searchErrors, fmt.Sprintf("official: %v", err))
				} else {
					for _, pkg := range limitResults(pkgs, searchLimit) {
						results = append(results, SearchResult{
							Name:        pkg.Name,
							Version:     pkg.Version,
							Description: pkg.Description,
							Source:      "official",
							Installed:   officialBackend.IsInstalled(pkg.Name),
						})
					}
				}
			}
		}

		// Search AUR (only if enabled in config and on Arch-based system)
		if (searchAll || searchAUR) && cfg.AUR.Enabled && backend.IsArchBased() {
			// Use config AUR URL if specified, otherwise default
			var aurBackend *aur.Backend
			if cfg.Mirrors.AUR != "" {
				aurBackend = aur.NewWithURL(cfg.Mirrors.AUR, cacheDir)
			} else {
				aurBackend = aur.New(cacheDir)
			}
			pkgs, err := aurBackend.Search(query)
			if err != nil {
				searchErrors = append(searchErrors, fmt.Sprintf("aur: %v", err))
			} else {
				for _, pkg := range limitResults(pkgs, searchLimit) {
					installed := false
					if officialBackend != nil {
						installed = officialBackend.IsInstalled(pkg.Name)
					}
					results = append(results, SearchResult{
						Name:        pkg.Name,
						Version:     pkg.Version,
						Description: pkg.Description,
						Source:      "aur",
						Installed:   installed,
					})
				}
			}
		}

		// Search ShedOS repository
		if searchAll || searchShedOS {
			timeout := 30 * time.Second
			if cfg.Network.Timeout > 0 {
				timeout = time.Duration(cfg.Network.Timeout) * time.Second
			}
			shedBackend := shedrepo.New(cacheDir, timeout)
			shedInstaller := installer.NewShedInstaller()
			pkgs, err := shedBackend.Search(query)
			if err != nil {
				searchErrors = append(searchErrors, fmt.Sprintf("shedos: %v", err))
			} else {
				for _, pkg := range limitResults(pkgs, searchLimit) {
					results = append(results, SearchResult{
						Name:        pkg.Name,
						Version:     pkg.Version,
						Description: pkg.Description,
						Source:      "shedos",
						Installed:   shedInstaller.IsInstalled(pkg.Name),
					})
				}
			}
		}

		// Search installed packages (both official backend AND .shed packages)
		if searchAll || searchInstalled {
			filtered := make([]SearchResult, 0)

			// Search official backend installed packages
			if officialBackend != nil {
				pkgs, err := officialBackend.GetInstalledPackages()
				if err != nil {
					searchErrors = append(searchErrors, fmt.Sprintf("installed/official: %v", err))
				} else {
					for _, pkg := range pkgs {
						if strings.Contains(strings.ToLower(pkg.Name), strings.ToLower(query)) ||
							strings.Contains(strings.ToLower(pkg.Description), strings.ToLower(query)) {
							filtered = append(filtered, SearchResult{
								Name:        pkg.Name,
								Version:     pkg.Version,
								Description: pkg.Description,
								Source:      "installed",
								Installed:   true,
							})
						}
					}
				}
			}

			// Search .shed installed packages (with full info)
			shedInstaller := installer.NewShedInstaller()
			shedPkgs, err := shedInstaller.ListInstalledWithInfo()
			if err != nil {
				searchErrors = append(searchErrors, fmt.Sprintf("installed/shed: %v", err))
			} else {
				for _, pkg := range shedPkgs {
					if strings.Contains(strings.ToLower(pkg.Name), strings.ToLower(query)) ||
						strings.Contains(strings.ToLower(pkg.Description), strings.ToLower(query)) {
						filtered = append(filtered, SearchResult{
							Name:        pkg.Name,
							Version:     pkg.Version,
							Description: pkg.Description,
							Source:      "installed/shed",
							Installed:   true,
						})
					}
				}
			}

			// Apply limit
			if searchLimit > 0 && len(filtered) > searchLimit {
				filtered = filtered[:searchLimit]
			}
			results = append(results, filtered...)
		}

		// Check if all sources failed
		if len(results) == 0 && len(searchErrors) > 0 {
			for _, e := range searchErrors {
				output.Warning("Search failed: %s", e)
			}
			return fmt.Errorf("no results found (all sources failed)")
		}

		// Output results
		if searchJSON {
			return outputJSON(cmd, results)
		}

		return outputFormatted(cmd, results, cfg)
	},
}

// limitResults limits the number of results if limit > 0
func limitResults(pkgs []pkgdb.PackageInfo, limit int) []pkgdb.PackageInfo {
	if limit <= 0 || len(pkgs) <= limit {
		return pkgs
	}
	return pkgs[:limit]
}

// outputJSON outputs results as JSON
func outputJSON(cmd *cobra.Command, results []SearchResult) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}
	cmd.Println(string(data))
	return nil
}

// outputFormatted outputs results with formatting and colors
func outputFormatted(cmd *cobra.Command, results []SearchResult, cfg *config.Config) error {
	if len(results) == 0 {
		cmd.Println("No packages found.")
		return nil
	}

	for _, r := range results {
		// Format: 📦 name         version    [source]
		name := fmt.Sprintf("%-20s", r.Name)
		version := fmt.Sprintf("%-12s", r.Version)
		source := fmt.Sprintf("[%s]", r.Source)

		// Add installed marker
		installed := ""
		if r.Installed && r.Source != "installed" {
			installed = " ✓"
		}

		if cfg.General.Color {
			// Color-code by source
			var coloredSource string
			switch r.Source {
			case "official":
				coloredSource = output.Colorize(output.Cyan, source)
			case "aur":
				coloredSource = output.Colorize(output.Yellow, source)
			case "shedos":
				coloredSource = output.Colorize(output.Green, source)
			case "installed", "installed/shed":
				coloredSource = output.Colorize(output.Magenta, source)
			default:
				coloredSource = source
			}
			cmd.Printf(" 📦 %s %s %s%s\n", name, version, coloredSource, installed)
		} else {
			cmd.Printf(" 📦 %s %s %s%s\n", name, version, source, installed)
		}
	}

	cmd.Printf("\nFound %d package(s)\n", len(results))
	return nil
}

// GetSearchCmd returns the search command for testing
func GetSearchCmd() *cobra.Command {
	return searchCmd
}

func init() {
	searchCmd.Flags().BoolVar(&searchOfficial, "official", false, "Search official repositories only")
	searchCmd.Flags().BoolVar(&searchAUR, "aur", false, "Search AUR only")
	searchCmd.Flags().BoolVar(&searchShedOS, "shedos", false, "Search ShedOS repository only")
	searchCmd.Flags().BoolVar(&searchInstalled, "installed", false, "Search installed packages only")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 0, "Limit results per source (0 = unlimited)")

	rootCmd.AddCommand(searchCmd)
}
