// Package pacman provides progress parsing for pacman output.
package alpm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// ProgressType indicates the type of progress event
type ProgressType int

const (
	ProgressDownloading ProgressType = iota
	ProgressInstalling
	ProgressUpgrading
	ProgressRemoving
	ProgressResolving
	ProgressChecking
	ProgressComplete
)

// String returns a string representation
func (pt ProgressType) String() string {
	switch pt {
	case ProgressDownloading:
		return "downloading"
	case ProgressInstalling:
		return "installing"
	case ProgressUpgrading:
		return "upgrading"
	case ProgressRemoving:
		return "removing"
	case ProgressResolving:
		return "resolving"
	case ProgressChecking:
		return "checking"
	case ProgressComplete:
		return "complete"
	default:
		return "unknown"
	}
}

// ProgressEvent represents a single progress update
type ProgressEvent struct {
	Type       ProgressType
	Package    string
	Message    string
	Current    int
	Total      int
	Percentage int
	Speed      string
	Downloaded string
	TotalSize  string
}

// ProgressCallback is called when progress events occur
type ProgressCallback func(ProgressEvent)

// Progress parses pacman output and emits progress events
type Progress struct {
	callback       ProgressCallback
	downloadRegex  *regexp.Regexp
	installRegex   *regexp.Regexp
	upgradeRegex   *regexp.Regexp
	percentRegex   *regexp.Regexp
	speedRegex     *regexp.Regexp
	progressRegex  *regexp.Regexp
	currentPkg     string
	totalPackages  int
	currentPackage int
}

// NewProgress creates a new progress parser
func NewProgress(callback ProgressCallback) *Progress {
	return &Progress{
		callback:      callback,
		downloadRegex: regexp.MustCompile(`(?i)downloading\s+(.+?)\.\.\.`),
		installRegex:  regexp.MustCompile(`(?i)installing\s+(.+?)\.\.\.`),
		upgradeRegex:  regexp.MustCompile(`(?i)upgrading\s+(.+?)\.\.\.`),
		percentRegex:  regexp.MustCompile(`\((\d+)/(\d+)\)\s+(\d+)%`),
		speedRegex:    regexp.MustCompile(`(\d+\.?\d*)\s*(B|KiB|MiB|GiB)/s`),
		progressRegex: regexp.MustCompile(`(\d+\.?\d*)\s*(B|KiB|MiB|GiB)\s*/\s*(\d+\.?\d*)\s*(B|KiB|MiB|GiB)`),
	}
}

// SetTotalPackages sets the expected total number of packages
func (p *Progress) SetTotalPackages(total int) {
	p.totalPackages = total
}

// ParseLine parses a line of pacman output and emits progress events
func (p *Progress) ParseLine(line string) {
	if p.callback == nil {
		return
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	event := ProgressEvent{
		Total:   p.totalPackages,
		Current: p.currentPackage,
	}

	// Parse percentage if present
	if matches := p.percentRegex.FindStringSubmatch(line); len(matches) > 3 {
		event.Current, _ = strconv.Atoi(matches[1])
		event.Total, _ = strconv.Atoi(matches[2])
		event.Percentage, _ = strconv.Atoi(matches[3])
		p.currentPackage = event.Current
		p.totalPackages = event.Total
	}

	// Parse speed if present
	if matches := p.speedRegex.FindStringSubmatch(line); len(matches) > 2 {
		event.Speed = matches[1] + " " + matches[2] + "/s"
	}

	// Parse download progress (e.g., "5.2 MiB / 10.4 MiB")
	if matches := p.progressRegex.FindStringSubmatch(line); len(matches) > 4 {
		event.Downloaded = matches[1] + " " + matches[2]
		event.TotalSize = matches[3] + " " + matches[4]
	}

	// Check for downloading
	if matches := p.downloadRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressDownloading
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Downloading %s...", matches[1])
		p.currentPkg = matches[1]
		p.callback(event)
		return
	}

	// Check for installing
	if matches := p.installRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressInstalling
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Installing %s...", matches[1])
		p.currentPkg = matches[1]
		p.callback(event)
		return
	}

	// Check for upgrading
	if matches := p.upgradeRegex.FindStringSubmatch(line); len(matches) > 1 {
		event.Type = ProgressUpgrading
		event.Package = matches[1]
		event.Message = fmt.Sprintf("Upgrading %s...", matches[1])
		p.currentPkg = matches[1]
		p.callback(event)
		return
	}

	// Check for removing
	if strings.Contains(strings.ToLower(line), "removing") {
		event.Type = ProgressRemoving
		event.Package = p.currentPkg
		event.Message = line
		p.callback(event)
		return
	}

	// Check for resolving dependencies
	if strings.Contains(strings.ToLower(line), "resolving") {
		event.Type = ProgressResolving
		event.Message = line
		p.callback(event)
		return
	}

	// Check for checking
	if strings.Contains(strings.ToLower(line), "checking") {
		event.Type = ProgressChecking
		event.Message = line
		p.callback(event)
		return
	}
}
