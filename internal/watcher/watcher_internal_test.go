package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
)

func startWatcher(t *testing.T) (string, string, <-chan string) {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join(root, "src")
	staging := filepath.Join(root, "staging")
	for _, dir := range []string{src, staging} {
		if err := os.Mkdir(dir, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	w, err := NewWatcher(&config.Config{Uploader: config.Uploader{Directory: src, Extensions: []string{".fits"}}})
	if err != nil {
		t.Fatal(err)
	}
	w.quiet = 50 * time.Millisecond
	uploads := make(chan string, 10)
	w.SetUploadCallback(func(path string) { uploads <- path })
	if err := w.Add(src); err != nil {
		t.Fatal(err)
	}
	go func() { _ = w.Start() }()
	t.Cleanup(func() { _ = w.Stop() })
	return src, staging, uploads
}

func expectUpload(t *testing.T, uploads <-chan string, want string) {
	t.Helper()
	select {
	case got := <-uploads:
		if got != want {
			t.Fatalf("uploaded %s, want %s", got, want)
		}
	case <-time.After(3 * time.Second):
		t.Fatalf("%s was never uploaded", want)
	}
}

func TestFileMovedInIsUploaded(t *testing.T) {
	t.Parallel()
	src, staging, uploads := startWatcher(t)
	if err := os.WriteFile(filepath.Join(staging, "light.fits"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(staging, "light.fits"), filepath.Join(src, "light.fits")); err != nil {
		t.Fatal(err)
	}
	expectUpload(t, uploads, filepath.Join(src, "light.fits"))
}

func TestFileWrittenIsUploaded(t *testing.T) {
	t.Parallel()
	src, _, uploads := startWatcher(t)
	if err := os.WriteFile(filepath.Join(src, "light.fits"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectUpload(t, uploads, filepath.Join(src, "light.fits"))
}

func TestDirectoryMovedInIsUploaded(t *testing.T) {
	t.Parallel()
	src, staging, uploads := startWatcher(t)
	night := filepath.Join(staging, "night")
	if err := os.Mkdir(night, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(night, "light.fits"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(night, filepath.Join(src, "night")); err != nil {
		t.Fatal(err)
	}
	expectUpload(t, uploads, filepath.Join(src, "night", "light.fits"))

	if err := os.WriteFile(filepath.Join(src, "night", "dark.fits"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	expectUpload(t, uploads, filepath.Join(src, "night", "dark.fits"))
}
