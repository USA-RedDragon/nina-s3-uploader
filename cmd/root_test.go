package cmd_test

import (
	"errors"
	"testing"

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
