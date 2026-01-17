package core

import (
	"errors"
	"fmt"
)

// ConflictSeverity represents how severe a conflict is
type ConflictSeverity int

const (
	ConflictError   ConflictSeverity = iota // Must be resolved before install
	ConflictWarning                         // Can proceed with caution
)

// Conflict represents a conflict between two packages
type Conflict struct {
	Package1   string
	Package2   string
	Reason     string
	Severity   ConflictSeverity
	Suggestion string // Suggested resolution
}

// String returns a human-readable description of the conflict
func (c Conflict) String() string {
	return fmt.Sprintf("%s conflicts with %s: %s", c.Package1, c.Package2, c.Reason)
}

// ConflictResult combines package and file conflicts
type ConflictResult struct {
	PackageConflicts []Conflict
	FileConflicts    []FileConflict
}

// HasErrors returns true if there are any error-level conflicts
func (cr *ConflictResult) HasErrors() bool {
	for _, c := range cr.PackageConflicts {
		if c.Severity == ConflictError {
			return true
		}
	}
	return len(cr.FileConflicts) > 0
}

// HasWarnings returns true if there are any warning-level conflicts
func (cr *ConflictResult) HasWarnings() bool {
	for _, c := range cr.PackageConflicts {
		if c.Severity == ConflictWarning {
			return true
		}
	}
	return false
}

// IsEmpty returns true if there are no conflicts
func (cr *ConflictResult) IsEmpty() bool {
	return len(cr.PackageConflicts) == 0 && len(cr.FileConflicts) == 0
}

// ErrorCount returns the number of error-level conflicts
func (cr *ConflictResult) ErrorCount() int {
	count := len(cr.FileConflicts) // All file conflicts are errors
	for _, c := range cr.PackageConflicts {
		if c.Severity == ConflictError {
			count++
		}
	}
	return count
}

// ErrConflictsFound is returned when conflicts are detected
var ErrConflictsFound = errors.New("conflicts found")

// ConflictDetector detects conflicts between packages
type ConflictDetector struct {
	db          PackageDB
	installedDB PackageDB // For checking conflicts with installed packages
}

// NewConflictDetector creates a new ConflictDetector
func NewConflictDetector(db PackageDB) *ConflictDetector {
	return &ConflictDetector{db: db}
}

// NewConflictDetectorWithInstalled creates a ConflictDetector that also checks installed packages
func NewConflictDetectorWithInstalled(db, installedDB PackageDB) *ConflictDetector {
	return &ConflictDetector{db: db, installedDB: installedDB}
}

// Detect checks for conflicts among the given packages
func (cd *ConflictDetector) Detect(packages []string) []Conflict {
	return cd.detectConflicts(packages, nil)
}

// DetectWithInstalled checks for conflicts including against installed packages
func (cd *ConflictDetector) DetectWithInstalled(packages []string, installedPackages []string) []Conflict {
	return cd.detectConflicts(packages, installedPackages)
}

// detectConflicts is the internal implementation
func (cd *ConflictDetector) detectConflicts(packages []string, installedPackages []string) []Conflict {
	var conflicts []Conflict
	pkgInfos := make(map[string]*PackageInfo)
	provides := make(map[string]string) // provided name -> package that provides it

	for _, name := range packages {
		info, _ := cd.db.GetInfo(name)
		if info != nil {
			pkgInfos[name] = info
		}
	}

	installedInfos := make(map[string]*PackageInfo)
	if cd.installedDB != nil && len(installedPackages) > 0 {
		for _, name := range installedPackages {
			info, _ := cd.installedDB.GetInfo(name)
			if info != nil {
				installedInfos[name] = info
			}
		}
	}

	for _, name := range packages {
		info := pkgInfos[name]
		if info == nil {
			continue
		}

		for _, conflictSpec := range info.Conflicts {
			// Parse version constraint from conflict spec
			conflictReq := ParseRequest(conflictSpec)
			conflictName := conflictReq.Name

			if targetInfo, exists := pkgInfos[conflictName]; exists {
				if conflictReq.Version != "" && conflictReq.Operator != "" {
					if !MatchesVersionConstraint(targetInfo.Version, conflictReq.Version, conflictReq.Operator) {
						continue // Version doesn't match, no conflict
					}
				}
				conflicts = append(conflicts, Conflict{
					Package1:   name,
					Package2:   conflictName,
					Reason:     "explicit conflict",
					Severity:   ConflictError,
					Suggestion: fmt.Sprintf("Remove either %s or %s", name, conflictName),
				})
			}

			if targetInfo, exists := installedInfos[conflictName]; exists {
				if conflictReq.Version != "" && conflictReq.Operator != "" {
					if !MatchesVersionConstraint(targetInfo.Version, conflictReq.Version, conflictReq.Operator) {
						continue
					}
				}
				conflicts = append(conflicts, Conflict{
					Package1:   name,
					Package2:   conflictName,
					Reason:     "conflicts with installed package",
					Severity:   ConflictError,
					Suggestion: fmt.Sprintf("Uninstall %s first", conflictName),
				})
			}
		}

		for _, prov := range info.Provides {
			// Parse provides (may have version like "name=1.0")
			provReq := ParseRequest(prov)
			provName := provReq.Name

			if existingPkg, exists := provides[provName]; exists {
				conflicts = append(conflicts, Conflict{
					Package1:   existingPkg,
					Package2:   name,
					Reason:     fmt.Sprintf("both provide %s", provName),
					Severity:   ConflictError,
					Suggestion: fmt.Sprintf("Choose either %s or %s", existingPkg, name),
				})
			} else {
				provides[provName] = name
			}
		}
	}

	for instName, instInfo := range installedInfos {
		for _, conflictSpec := range instInfo.Conflicts {
			conflictReq := ParseRequest(conflictSpec)
			conflictName := conflictReq.Name

			if targetInfo, exists := pkgInfos[conflictName]; exists {
				if conflictReq.Version != "" && conflictReq.Operator != "" {
					if !MatchesVersionConstraint(targetInfo.Version, conflictReq.Version, conflictReq.Operator) {
						continue
					}
				}
				conflicts = append(conflicts, Conflict{
					Package1:   instName,
					Package2:   conflictName,
					Reason:     "installed package conflicts with new package",
					Severity:   ConflictError,
					Suggestion: fmt.Sprintf("Uninstall %s first, or skip installing %s", instName, conflictName),
				})
			}
		}
	}

	return conflicts
}

// DetectAll performs comprehensive conflict detection returning unified result
func (cd *ConflictDetector) DetectAll(packages []string, installedPackages []string, fileChecker *FileConflictChecker) *ConflictResult {
	result := &ConflictResult{
		PackageConflicts: cd.DetectWithInstalled(packages, installedPackages),
	}

	if fileChecker != nil {
		result.FileConflicts = fileChecker.CheckConflicts()
	}

	return result
}
