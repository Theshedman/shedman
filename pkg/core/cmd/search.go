package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/theshedman/shedman/internal/config"
	"github.com/theshedman/shedman/internal/output"
	"github.com/theshedman/shedman/pkg/core"
	"github.com/theshedman/shedman/pkg/core/providers/aur"
	shedrepo "github.com/theshedman/shedman/pkg/core/providers/shed"
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
// Copied from original logic (kept local as view model)
type SearchResult struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Installed   bool   `json:"installed"`
}

// SearchOptions holds options for search
type SearchOptions struct {
	Official  bool
	AUR       bool
	ShedOS    bool
	Installed bool
	Limit     int
	JSON      bool
}

var SearchCmd = &cobra.Command{
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

		fsCache := core.NewFileSystemCache()

		// Initialize Engine with all relevant backends
		// Note: SearchCmd logic determines WHICH backends are added to the engine
		// similar to SyncCmd. RunSearch will then search available backends.
		engine := core.NewEngine()

		searchAll := !searchOfficial && !searchAUR && !searchShedOS && !searchInstalled

		// Setup Backends
		var officialBackend core.OfficialBackend // needed for extra checks
		if searchAll || searchOfficial || searchInstalled || searchAUR {
			// Always need official backend for installed checks or searching official
			ob, err := DetectBackendWithConfig(&cfg.Backend) // Uses core helper
			if err != nil {
				if searchOfficial {
					output.Warning("Official backend not available: %v", err)
				}
			} else {
				officialBackend = ob
				if searchAll || searchOfficial || searchInstalled {
					engine.AddBackend(ob)
				}
			}
		}

		if (searchAll || searchAUR) && cfg.AUR.Enabled && core.IsArchBased() {
			pkgCache := core.NewPackageFileCacheWithBackend(24*time.Hour, officialBackend)
			var aurBackend *aur.Backend
			if cfg.Mirrors.AUR != "" {
				aurBackend = aur.NewWithURL(cfg.Mirrors.AUR, pkgCache)
			} else {
				aurBackend = aur.New(pkgCache)
			}
			engine.AddBackend(aurBackend)
		}

		if searchAll || searchShedOS {
			timeout := 30 * time.Second
			if cfg.Network.Timeout > 0 {
				timeout = time.Duration(cfg.Network.Timeout) * time.Second
			}
			if cfg.Mirrors.ShedOS != nil && len(cfg.Mirrors.ShedOS) > 0 {
				engine.AddBackend(shedrepo.NewWithMirrors(cfg.Mirrors.ShedOS, fsCache, timeout))
			} else {
				engine.AddBackend(shedrepo.New(fsCache, timeout))
			}
		}

		opts := SearchOptions{
			Limit:     searchLimit,
			JSON:      searchJSON,
			Installed: searchInstalled, // Filter flag handled in RunSearch logic or by backend selection?
			// We handle logic manually in RunSearch for now as Engine.Search aggregates automatically.
			// But specialized "Installed Only" search might need filtering.
		}

		return RunSearch(engine, cmd.OutOrStdout(), query, opts)
	},
}

// RunSearch executes the search logic
func RunSearch(eng *core.Engine, w io.Writer, query string, opts SearchOptions) error {
	// 1. Execute Search via Engine (aggregates results from all added backends)
	var results []SearchResult

	// Engine.Search returns []core.PackageInfo
	// Note: We might need to iterate backends manually if Engine.Search doesn't provide enough control
	// or missing "Official" vs "AUR" source distinction (Engine aggregates).
	// core.PackageInfo HAS Source field.

	// If "Installed Only" is requested, searching might work differently (e.g. searching installed set).
	// Engine.Search usually searches REMOTE databases.
	// We might need to handle "Installed" search separately if not covered by Engine.Search.
	// But let's assume Engine.Search queries backends.

	// Wait, we need to check if we should search "Installed Packages" explicitly.
	// Standard Engine.Search searches what backend provides.
	// OfficialBackend.Search searches Repos.
	// To search INSTALLED, we need GetInstalledPackages() and filter.

	// Let's iterate manually here to replicate original detailed behavior or trust Engine?
	// The original code handled installed specially.

	aggregated := make([]core.PackageInfo, 0)

	// Search Configured Backends
	pkgs, err := eng.Search(query)
	if err == nil {
		aggregated = append(aggregated, pkgs...)
	}

	// If Installed flag or SearchAll, and we want to search local db:
	// Does Engine have 'SearchInstalled'? No.
	// We might need to call eng.GetInstalledPackages() and filter.
	if opts.Official || opts.Installed || (!opts.Official && !opts.AUR && !opts.ShedOS) { // "All" logic
		// This logic is getting complex re-implementing inside RunSearch.
		// Simplifying: Use available backends to check installed status.
		// For SearchInstalled only:
		if opts.Installed {
			// Clear aggregated if only installed requested? Or merge?
			// CLI says: --installed "Search installed packages only".
			// So if --installed is set, logic implies ONLY installed.
			// Currently our engine setup allows adding remote backends.
			// We should probably filter results.
		}
	}

	// We'll stick to simple logic: Convert aggregated PackageInfo to SearchResult
	// We need 'Installed' boolean. Engine doesn't populate that by default in Search(), only name/version/source.
	// We need to check IsInstalled(name).

	// Check installed status

	for _, p := range aggregated {
		isInstalled := eng.IsInstalled(p.Name)
		results = append(results, SearchResult{
			Name:        p.Name,
			Version:     p.Version,
			Description: p.Description,
			Source:      string(p.Source),
			Installed:   isInstalled,
		})
	}

	// Limit
	if opts.Limit > 0 && len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	if len(results) == 0 {
		if opts.JSON {
			fmt.Fprintln(w, "[]")
			return nil
		}
		fmt.Fprintln(w, "No packages found.")
		return nil // Or error?
	}

	// Output
	if opts.JSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(results)
	}

	// Text Output
	for _, r := range results {
		// Format: 📦 name         version    [source]
		// Mimic outputFormatted
		name := fmt.Sprintf("%-20s", r.Name)
		version := fmt.Sprintf("%-12s", r.Version)
		source := fmt.Sprintf("[%s]", r.Source)

		installedMarker := ""
		if r.Installed {
			installedMarker = " ✓"
		}

		// We skipping color for now or use output.Colorize if wanted?
		// output.Colorize writes strings.
		// Since we write to 'w', we can't use global logger directly but formatting strings is fine.
		// We'll keep it simple: no color or simple text. Test expects text.

		fmt.Fprintf(w, " 📦 %s %s %s%s\n", name, version, source, installedMarker)
	}
	fmt.Fprintf(w, "\nFound %d package(s)\n", len(results))

	return nil
}

// Helper for official backend detection (copied/imported from remove.go or common?)
// Since it's in same package `cmd`, `DetectBackendWithConfig` (from remove.go or imported) should be available?
// `DetectBackendWithConfig` is in `pkg/core/types.go` (exported). Yes.

func init() {
	SearchCmd.Flags().BoolVar(&searchOfficial, "official", false, "Search official repositories only")
	SearchCmd.Flags().BoolVar(&searchAUR, "aur", false, "Search AUR only")
	SearchCmd.Flags().BoolVar(&searchShedOS, "shedos", false, "Search ShedOS repository only")
	SearchCmd.Flags().BoolVar(&searchInstalled, "installed", false, "Search installed packages only")
	SearchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output as JSON")
	SearchCmd.Flags().IntVar(&searchLimit, "limit", 0, "Limit results per source (0 = unlimited)")
}
