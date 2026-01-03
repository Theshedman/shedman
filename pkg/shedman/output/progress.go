package output

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"
)

// ProgressBar represents a terminal progress bar
type ProgressBar struct {
	total     int64
	current   int64
	width     int
	label     string
	startTime time.Time
}

// NewProgressBar creates a new progress bar
func NewProgressBar(total int64, label string) *ProgressBar {
	return &ProgressBar{
		total:     total,
		current:   0,
		width:     getProgressBarWidth(),
		label:     label,
		startTime: time.Now(),
	}
}

// getProgressBarWidth returns appropriate width based on terminal
func getProgressBarWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || width < 60 {
		return 30 // Default for narrow terminals
	}
	// Use about 40% of terminal width for the bar, max 50
	barWidth := width * 40 / 100
	if barWidth > 50 {
		barWidth = 50
	}
	if barWidth < 20 {
		barWidth = 20
	}
	return barWidth
}

// SetTotal updates the total value
func (p *ProgressBar) SetTotal(total int64) {
	p.total = total
}

// Update updates the progress bar with current value
func (p *ProgressBar) Update(current int64) {
	p.current = current
	p.render()
}

// Increment adds to the current progress
func (p *ProgressBar) Increment(delta int64) {
	p.current += delta
	p.render()
}

// Complete marks the progress as done
func (p *ProgressBar) Complete() {
	p.current = p.total
	p.render()
	fmt.Println() // Newline after completion
}

// render draws the progress bar with ETA
func (p *ProgressBar) render() {
	if p.total <= 0 {
		return
	}

	percent := float64(p.current) / float64(p.total) * 100
	filled := int(float64(p.width) * float64(p.current) / float64(p.total))
	empty := p.width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)

	// Calculate speed and ETA
	elapsed := time.Since(p.startTime).Seconds()
	speed := float64(p.current) / elapsed / 1024 // KB/s

	// Calculate ETA
	eta := ""
	if p.current > 0 && p.current < p.total {
		remaining := float64(p.total-p.current) / (float64(p.current) / elapsed)
		eta = formatDuration(time.Duration(remaining) * time.Second)
	} else if p.current >= p.total {
		eta = "done"
	}

	// Clear line and print
	fmt.Fprintf(os.Stdout, "\r%s [%s] %.1f%% %.1f KB/s ETA: %s  ",
		p.label, Colorize(Cyan, bar), percent, speed, eta)
}

// formatDuration formats duration as mm:ss or hh:mm:ss
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

// Spinner represents an animated spinner with thread safety
type Spinner struct {
	frames  []string
	current int
	label   string
	done    chan bool
	running bool
	mu      sync.Mutex
}

// NewSpinner creates a new spinner
func NewSpinner(label string) *Spinner {
	return &Spinner{
		frames: []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		label:  label,
		done:   make(chan bool, 1), // Buffered channel
	}
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	go func() {
		for {
			select {
			case <-s.done:
				return
			default:
				s.mu.Lock()
				if !s.running {
					s.mu.Unlock()
					return
				}
				frame := s.frames[s.current%len(s.frames)]
				s.current++
				s.mu.Unlock()

				fmt.Fprintf(os.Stdout, "\r%s %s", Colorize(Cyan, frame), s.label)
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
}

// Stop stops the spinner
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}
	s.running = false

	select {
	case s.done <- true:
	default:
	}

	fmt.Fprintf(os.Stdout, "\r%s\r", strings.Repeat(" ", len(s.label)+3))
}

// StopWithSuccess stops the spinner with a success message
func (s *Spinner) StopWithSuccess(msg string) {
	s.Stop()
	Success("%s", msg)
}

// StopWithError stops the spinner with an error message
func (s *Spinner) StopWithError(msg string) {
	s.Stop()
	Error("%s", msg)
}

// UpdateLabel changes the spinner label
func (s *Spinner) UpdateLabel(label string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.label = label
}

// DownloadProgress shows download progress with ETA
func DownloadProgress(label string, downloaded, total int64, speed float64) {
	percent := float64(downloaded) / float64(total) * 100
	if total <= 0 {
		fmt.Fprintf(os.Stdout, "\r%s %.2f MB (%.1f KB/s)",
			label, float64(downloaded)/1024/1024, speed)
		return
	}

	// Calculate ETA
	eta := ""
	if speed > 0 && downloaded < total {
		remaining := float64(total-downloaded) / (speed * 1024)
		eta = formatDuration(time.Duration(remaining) * time.Second)
	}

	fmt.Fprintf(os.Stdout, "\r%s %.1f%% (%.1f KB/s) ETA: %s  ",
		label, percent, speed, eta)
}
