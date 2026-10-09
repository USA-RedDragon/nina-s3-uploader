package cmd_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/USA-RedDragon/configulator/v2"
	"github.com/USA-RedDragon/nina-s3-uploader/cmd"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
)

func TestInvalidLogLevelIsAnError(t *testing.T) {
	t.Parallel()
	c := cmd.NewCommand("test", "test")
	c.SetArgs([]string{
		"--log-level", "bogus",
		"--s3.bucket", "bucket",
		"--uploader.directory", "src",
		"--uploader.extensions", ".fits",
		"--uploader.local.directory", "local",
	})
	err := c.Execute()
	if !errors.Is(err, config.ErrInvalidLogLevel) {
		t.Fatalf("expected ErrInvalidLogLevel, got %v", err)
	}
}

func TestConfigEnvVarPicksTheFile(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.yaml")
	t.Setenv("CONFIG", missing)
	c := cmd.NewCommand("test", "test")
	c.SetArgs([]string{})
	err := c.Execute()
	var missingErr *configulator.MissingFileError
	if !errors.As(err, &missingErr) || missingErr.Path != missing {
		t.Fatalf("expected a MissingFileError for %s, got %v", missing, err)
	}
}
