package core

import (
	"strconv"
	"strings"
	"unicode"
)

// Pre-release tag priorities (lower = older)
var preReleasePriority = map[string]int{
	"alpha": 1,
	"a":     1,
	"beta":  2,
	"b":     2,
	"rc":    3,
	"pre":   0,
	"dev":   0,
}

// CompareVersions compares two version strings following semantic versioning
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	p1 := parseVersion(v1)
	p2 := parseVersion(v2)

	// Compare epochs first
	if p1.Epoch != p2.Epoch {
		if p1.Epoch < p2.Epoch {
			return -1
		}
		return 1
	}

	// Compare numeric parts
	maxLen := len(p1.Parts)
	if len(p2.Parts) > maxLen {
		maxLen = len(p2.Parts)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(p1.Parts) {
			n1 = p1.Parts[i]
		}
		if i < len(p2.Parts) {
			n2 = p2.Parts[i]
		}

		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	// Compare pre-release tags
	// A version with pre-release is LESS than the same version without
	// e.g., 1.0.0-alpha < 1.0.0
	if p1.PreRelease != "" && p2.PreRelease == "" {
		return -1
	}
	if p1.PreRelease == "" && p2.PreRelease != "" {
		return 1
	}
	if p1.PreRelease != "" && p2.PreRelease != "" {
		return comparePreRelease(p1.PreRelease, p2.PreRelease)
	}

	// Compare package release numbers
	if p1.PkgRel != p2.PkgRel {
		if p1.PkgRel < p2.PkgRel {
			return -1
		}
		return 1
	}

	return 0
}

// parsedVersion holds the parsed components of a version string
type parsedVersion struct {
	Epoch      int
	Parts      []int
	PreRelease string
	PkgRel     int
}

// parseVersion parses a version string into its components
// Format: [epoch:]major.minor.patch[-prerelease][-pkgrel]
func parseVersion(v string) parsedVersion {
	pv := parsedVersion{}

	// Remove common prefixes
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")

	// Extract epoch (e.g., "1:2.0.0")
	if idx := strings.Index(v, ":"); idx != -1 {
		epoch, _ := strconv.Atoi(v[:idx])
		pv.Epoch = epoch
		v = v[idx+1:]
	}

	// Split on hyphen to separate pre-release and pkg release
	// e.g., "1.0.0-alpha-1" -> parts: "1.0.0", prerel: "alpha", pkgrel: "1"
	// e.g., "1.0.0-1" -> parts: "1.0.0", pkgrel: "1"
	hyphenParts := strings.Split(v, "-")
	mainVersion := hyphenParts[0]

	// Parse remaining parts for pre-release and pkg release
	for i := 1; i < len(hyphenParts); i++ {
		part := hyphenParts[i]
		// Check if it's a pre-release tag
		if isPreRelease(part) {
			pv.PreRelease = strings.ToLower(part)
		} else if isNumeric(part) {
			// Numeric part after hyphen is package release
			pkgRel, _ := strconv.Atoi(part)
			pv.PkgRel = pkgRel
		} else {
			// Treat as pre-release if not purely numeric
			pv.PreRelease = strings.ToLower(part)
		}
	}

	// Parse main version parts (e.g., "1.2.3" -> [1, 2, 3])
	pv.Parts = parseVersionParts(mainVersion)

	return pv
}

// isPreRelease checks if a string looks like a pre-release tag
func isPreRelease(s string) bool {
	lower := strings.ToLower(s)
	for tag := range preReleasePriority {
		if strings.HasPrefix(lower, tag) {
			return true
		}
	}
	return false
}

// isNumeric checks if a string is purely numeric
func isNumeric(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return len(s) > 0
}

// comparePreRelease compares two pre-release strings
// e.g., "alpha" < "beta" < "rc" < "rc2"
func comparePreRelease(p1, p2 string) int {
	// Extract the tag and number from each
	tag1, num1 := splitPreRelease(p1)
	tag2, num2 := splitPreRelease(p2)

	// Get priorities
	pri1 := preReleasePriority[tag1]
	pri2 := preReleasePriority[tag2]

	if pri1 != pri2 {
		if pri1 < pri2 {
			return -1
		}
		return 1
	}

	// Same tag, compare numbers (e.g., rc1 < rc2)
	if num1 != num2 {
		if num1 < num2 {
			return -1
		}
		return 1
	}

	return 0
}

// splitPreRelease splits "alpha1" into ("alpha", 1)
func splitPreRelease(pr string) (string, int) {
	pr = strings.ToLower(pr)

	// Find where digits start
	i := len(pr)
	for j := len(pr) - 1; j >= 0; j-- {
		if !unicode.IsDigit(rune(pr[j])) {
			break
		}
		i = j
	}

	tag := pr[:i]
	num := 0
	if i < len(pr) {
		num, _ = strconv.Atoi(pr[i:])
	}

	return tag, num
}

// parseVersionParts splits version like "1.2.3" into [1, 2, 3]
func parseVersionParts(v string) []int {
	// Replace common separators with dots
	v = strings.ReplaceAll(v, "_", ".")

	parts := strings.Split(v, ".")
	result := make([]int, 0, len(parts))

	for _, p := range parts {
		// Extract numeric prefix from part (e.g., "3rc1" -> 3)
		numStr := ""
		for _, c := range p {
			if c >= '0' && c <= '9' {
				numStr += string(c)
			} else {
				break
			}
		}
		if numStr != "" {
			num, _ := strconv.Atoi(numStr)
			result = append(result, num)
		}
	}

	return result
}

// MatchesVersionConstraint checks if a version satisfies a constraint
func MatchesVersionConstraint(version, constraint, operator string) bool {
	if constraint == "" || operator == "" {
		return true // No constraint = any version matches
	}

	cmp := CompareVersions(version, constraint)

	switch operator {
	case OpEqual:
		return cmp == 0
	case OpGreaterEqual:
		return cmp >= 0
	case OpLessEqual:
		return cmp <= 0
	case OpGreater:
		return cmp > 0
	case OpLess:
		return cmp < 0
	default:
		return true
	}
}

// RequestMatchesPackage checks if a package satisfies the request's version constraint
func RequestMatchesPackage(req Request, packageVersion string) bool {
	return MatchesVersionConstraint(packageVersion, req.Version, req.Operator)
}
