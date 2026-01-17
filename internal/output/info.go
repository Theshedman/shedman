package output

import (
	"fmt"
)

// PrintInfoKV prints a Key-Value pair for package information.
// It aligns keys and applies standard formatting.
func PrintInfoKV(key, value string) {
	// Simple aligned output: Key : Value
	_, _ = fmt.Printf("%-18s : %s\n", key, value)

}
