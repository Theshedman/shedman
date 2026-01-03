package resolver

import (
	"strconv"
	"strings"
)

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	parts1 := normalizeVersion(v1)
	parts2 := normalizeVersion(v2)

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1 = parts1[i]
		}
		if i < len(parts2) {
			p2 = parts2[i]
		}

		if p1 < p2 {
			return -1
		}
		if p1 > p2 {
			return 1
		}
	}

	return 0
}

// normalizeVersion splits a version string into numeric parts
func normalizeVersion(v string) []int {
	// Remove common prefixes like "v" or epoch like "1:"
	v = strings.TrimPrefix(v, "v")
	if idx := strings.Index(v, ":"); idx != -1 {
		// Keep epoch as first part
		epoch, rest := v[:idx], v[idx+1:]
		epochNum, _ := strconv.Atoi(epoch)
		parts := []int{epochNum}
		parts = append(parts, parseVersionParts(rest)...)
		return parts
	}

	return parseVersionParts(v)
}

// parseVersionParts splits version like "1.2.3-4" into [1, 2, 3, 4]
func parseVersionParts(v string) []int {
	// Replace common separators with dots
	v = strings.ReplaceAll(v, "-", ".")
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
