package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTUICmd(t *testing.T) {
	assert.Equal(t, "tui", TUICmd.Use)
	assert.NotEmpty(t, TUICmd.Short)
	assert.NotNil(t, TUICmd.RunE)
}
