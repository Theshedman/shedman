package cmd

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/theshedman/shedman/internal/output"
)

// ConfirmOptions aliases output.ConfirmOptions for test injection.
type ConfirmOptions = output.ConfirmOptions

type serviceEnabler interface {
	Enable(name string) error
}

var confirmServicePrompt = func(name string, opts ConfirmOptions) bool {
	prompt := fmt.Sprintf("Enable and start %s now?", name)
	return output.Confirm(prompt, opts)
}

func detectSystemdUnits(files []string) []string {
	var units []string
	for _, file := range files {
		if !strings.HasPrefix(file, "/usr/lib/systemd/system/") {
			continue
		}
		base := filepath.Base(file)
		if strings.HasSuffix(base, ".service") || strings.HasSuffix(base, ".socket") {
			units = append(units, base)
		}
	}
	return units
}

func promptEnableServices(w io.Writer, manager serviceEnabler, services []string) error {
	if len(services) == 0 {
		return nil
	}
	if manager == nil {
		return fmt.Errorf("service manager not available")
	}

	opts := ConfirmOptions{Default: false}
	for _, svc := range services {
		if !confirmServicePrompt(svc, opts) {
			continue
		}
		if err := manager.Enable(svc); err != nil {
			return err
		}
		if w != nil {
			_, _ = fmt.Fprintf(w, "Enabled %s.\n", svc)
		}
	}
	return nil
}

func dedupeStrings(items []string) []string {
	seen := make(map[string]bool)
	var out []string
	for _, item := range items {
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}
