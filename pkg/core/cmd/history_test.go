package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRunHistory(t *testing.T) {
	logData := `
[2023-01-01T10:00:00-0500] [ALPM] transaction started
[2023-01-01T10:00:01-0500] [ALPM] installed package1 (1.0-1)
[2023-01-02T10:00:00-0500] [ALPM] upgraded package2 (1.0 -> 2.0)
[2023-01-03T10:00:00-0500] [ALPM] removed package3 (1.0)
[2023-01-04T10:00:00-0500] [PACMAN] Running 'pacman -S foo'
`
	// Default time format in pacman log is [YYYY-MM-DDTHH:MM:SS-0700]

	tests := []struct {
		name         string
		limit        int
		json         bool
		since        string
		pkg          string
		wantContains []string
		wantExcludes []string
	}{
		{
			name:         "No Limit",
			limit:        0,
			wantContains: []string{"installed package1", "upgraded package2", "removed package3"},
		},
		{
			name:         "Limit 1",
			limit:        1,
			wantContains: []string{"removed package3"},
			wantExcludes: []string{"installed package1", "upgraded package2"},
		},
		{
			name:         "Filter Package",
			pkg:          "package1",
			wantContains: []string{"installed package1"},
			wantExcludes: []string{"package2", "package3"},
		},
		{
			name:         "Filter Since",
			since:        "2023-01-02",
			wantContains: []string{"upgraded package2", "removed package3"},
			wantExcludes: []string{"installed package1"},
		},
		{
			name:         "JSON Output",
			json:         true,
			wantContains: []string{`"action": "installed"`, `"package": "package1"`, `"version": "1.0-1"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := strings.NewReader(logData)
			var w bytes.Buffer

			// Simple date parsing for test convenience passed as string
			var sinceTime time.Time
			if tt.since != "" {
				sinceTime, _ = time.Parse("2006-01-02", tt.since)
			}

			opts := HistoryOptions{
				Limit:   tt.limit,
				JSON:    tt.json,
				Package: tt.pkg,
				Since:   sinceTime,
			}

			if err := RunHistory(r, &w, opts); err != nil {
				t.Errorf("RunHistory() error = %v", err)
			}

			output := w.String()
			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("Output missing %q. Got:\n%s", want, output)
				}
			}
			for _, exclude := range tt.wantExcludes {
				if strings.Contains(output, exclude) {
					t.Errorf("Output contains excluded %q. Got:\n%s", exclude, output)
				}
			}
		})
	}
}
