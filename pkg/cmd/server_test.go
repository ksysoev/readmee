package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_InitLoggerFails(t *testing.T) {
	flags := &cmdFlags{
		LogLevel: "WrongLogLevel",
	}

	err := RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to init logger")
}

func TestRunCommand_LoadConfigFails(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte("invalid config"), 0o600)
	require.NoError(t, err)

	flags := &cmdFlags{
		ConfigPath: configPath,
		LogLevel:   "info",
	}

	err = RunCommand(t.Context(), flags)
	assert.ErrorContains(t, err, "failed to load config:")
}
