package util

import (
	"os"
	"testing"
)

func TestGetEnvOrPrompt(t *testing.T) {
	const envKey = "SHEDMAN_TEST_KEY"

	// Case 1: Environment variable is set
	t.Run("EnvVarSet", func(t *testing.T) {
		t.Setenv(envKey, "env_secret")
		val := GetEnvOrPrompt(envKey, "Enter secret: ")
		if val != "env_secret" {
			t.Errorf("Expected 'env_secret', got '%s'", val)
		}
	})

	// Case 2: Environment variable is missing, fallback to prompt
	t.Run("PromptFallback", func(t *testing.T) {
		// Ensure env var is unset
		_ = os.Unsetenv(envKey)

		// Mock Stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		defer func() { os.Stdin = oldStdin }()
		os.Stdin = r

		// Write simulated input
		go func() {
			defer func() { _ = w.Close() }()

			_, _ = w.Write([]byte("prompt_secret\n"))

		}()

		val := GetEnvOrPrompt(envKey, "Enter secret: ")
		if val != "prompt_secret" {
			t.Errorf("Expected 'prompt_secret', got '%s'", val)
		}
	})
}
