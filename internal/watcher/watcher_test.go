package watcher_test

import (
	"path/filepath"
	"testing"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/watcher"
)

func TestBadDirsMatchSystemDirectories(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"System Volume Information", "lost+found", "$RECYCLE.BIN"} {
		matched := false
		for _, re := range watcher.BadDirs {
			if re.MatchString(name) {
				matched = true
			}
		}
		if !matched {
			t.Errorf("%q is not skipped", name)
		}
	}
}

func TestAddMissingDirectoryIsAnError(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "gone")
	w, err := watcher.NewWatcher(&config.Config{Uploader: config.Uploader{Directory: missing}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Stop() })
	if err := w.Add(missing); err == nil {
		t.Fatal("expected an error")
	}
}
