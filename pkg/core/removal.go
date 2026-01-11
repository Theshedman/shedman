package core

// InstalledPackage represents a package installed on the system
type InstalledPackage struct {
	Name    string
	Depends []string
}

// InstalledProvider interface for retrieving installed packages
type InstalledProvider interface {
	GetInstalledPackages() []InstalledPackage
}

// CalculateRecursiveRemoval calculates the list of packages to remove recursively
// equivalent to pacman -Rs (remove target + unneeded dependencies)
func CalculateRecursiveRemoval(targets []string, provider InstalledProvider) []string {
	allPkgs := provider.GetInstalledPackages()

	// Map to quickly look up package info by name
	pkgMap := make(map[string]InstalledPackage)
	for _, p := range allPkgs {
		pkgMap[p.Name] = p
	}

	// 1. Build Reverse Dependency Graph (RequiredBy)
	// Key: Package, Value: List of packages that depend on it
	requiredBy := make(map[string][]string)
	for _, p := range allPkgs {
		for _, dep := range p.Depends {
			requiredBy[dep] = append(requiredBy[dep], p.Name)
		}
	}

	// 2. Initialize removal set with targets
	toRemove := make(map[string]bool)
	for _, t := range targets {
		toRemove[t] = true
	}

	// 3. Iteratively look for orphans
	// An orphan is a dependency of a package in 'toRemove' that is NOT required by any package OUTSIDE of 'toRemove'
	changed := true
	for changed {
		changed = false

		// Candidate list: dependencies of current removal set
		candidates := make(map[string]bool)
		for name := range toRemove {
			if p, ok := pkgMap[name]; ok {
				for _, dep := range p.Depends {
					if !toRemove[dep] {
						candidates[dep] = true
					}
				}
			}
		}

		for cand := range candidates {
			// Check if candidate is required by anything NOT in 'toRemove'
			isNeeded := false
			requirers := requiredBy[cand]
			for _, req := range requirers {
				if !toRemove[req] {
					isNeeded = true
					break
				}
			}

			if !isNeeded {
				// It's an orphan! Remove it.
				toRemove[cand] = true
				changed = true
			}
		}
	}

	// Convert set to slice
	result := make([]string, 0, len(toRemove))
	for name := range toRemove {
		result = append(result, name)
	}

	return result
}
