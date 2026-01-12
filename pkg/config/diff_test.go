package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffer_CalculateHash(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shedman-diff-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	file := filepath.Join(tmpDir, "test.txt")
	content := []byte("hello world\n")
	err = os.WriteFile(file, content, 0644)
	require.NoError(t, err)

	d := NewDiffer()
	hash, err := d.CalculateHash(file)
	require.NoError(t, err)

	// echo -n "hello world\n" | sha256sum
	// a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447
	expected := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	assert.Equal(t, expected, hash)
}

func TestDiffer_UnifiedDiff(t *testing.T) {
	d := NewDiffer()

	oldContent := "line1\nline2\nline3\n"
	newContent := "line1\nline2 changed\nline3\nadded\n"

	diff, err := d.GenerateDiff("old.txt", oldContent, "new.txt", newContent)
	require.NoError(t, err)

	assert.Contains(t, diff, "--- old.txt")
	assert.Contains(t, diff, "+++ new.txt")
	assert.Contains(t, diff, "-line2")
	assert.Contains(t, diff, "+line2 changed")
	assert.Contains(t, diff, "+added")
}
