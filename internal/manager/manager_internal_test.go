package manager

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileLogsNoErrors(t *testing.T) { //nolint:paralleltest // replaces the default logger
	var logs bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelError})))
	t.Cleanup(func() { slog.SetDefault(prev) })

	dir := t.TempDir()
	src := filepath.Join(dir, "src.fits")
	dst := filepath.Join(dir, "dst.fits")
	if err := os.WriteFile(src, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "data" {
		t.Errorf("copied %q", got)
	}
	if logs.Len() != 0 {
		t.Errorf("unexpected error logs: %s", logs.String())
	}
}
