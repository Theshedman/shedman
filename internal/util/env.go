package util

import (
	"fmt"
	"os"

	"github.com/theshedman/shedman/internal/output"
)

// GetEnvOrPrompt checks for an environment variable.
// If found, it prints a message and returns the value.
// If not found, it prompts the user using output.ReadInput.
func GetEnvOrPrompt(envVar, promptMsg string) string {
	if val := os.Getenv(envVar); val != "" {
		_, _ = fmt.Printf("Using credentials from %s\n", envVar)

		return val
	}
	// Fallback to prompt
	val, err := output.ReadInput(promptMsg)
	if err != nil {
		return ""
	}
	return val
}
