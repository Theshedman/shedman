package log

import (
	"bufio"
	"os"
	"regexp"
	"time"
)

var (
	reLogLine = regexp.MustCompile(`^\[(.*?)\] \[ALPM\] (.*)$`)
)

// Parse parses a pacman log file
func Parse(path string) ([]Transaction, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var txs []Transaction
	var currentTx *Transaction

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := reLogLine.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		timestampStr := matches[1]
		message := matches[2]

		ts, err := time.Parse("2006-01-02T15:04:05-0700", timestampStr)
		if err != nil {
			continue
		}

		if message == "transaction started" {
			currentTx = &Transaction{
				ID:        ts.Format("20060102150405"),
				Timestamp: ts,
				Success:   false, // Will be set on complete
			}
		} else if message == "transaction completed" {
			if currentTx != nil {
				currentTx.Success = true
				// Determine main action based on packages
				if currentTx.Action == "" && len(currentTx.Packages) > 0 {
					// Heuristic: take action from first package entry if possible,
					// but we didn't store action per package in Transaction struct (just list of strings).
					// Better parsing needed for full fidelity, but for list view this is okay.
					// Let's rely on what we set during package parsing lines.
				}
				txs = append(txs, *currentTx)
				currentTx = nil
			}
		} else if currentTx != nil {
			// Inside transaction
			// Check for installed/removed/upgraded
			// "installed <pkg> (<ver>)"
			// "removed <pkg> (<ver>)"
			if len(message) > 10 && message[:10] == "installed " {
				currentTx.Packages = append(currentTx.Packages, message[10:])
				if currentTx.Action == "" {
					currentTx.Action = "installed"
				}
			} else if len(message) > 8 && message[:8] == "removed " {
				currentTx.Packages = append(currentTx.Packages, message[8:])
				if currentTx.Action == "" {
					currentTx.Action = "removed"
				}
			} else if len(message) > 9 && message[:9] == "upgraded " {
				currentTx.Packages = append(currentTx.Packages, message[9:])
				if currentTx.Action == "" {
					currentTx.Action = "upgraded"
				}
			}
		}
	}

	return txs, scanner.Err()
}
