package watcher_test

import (
	"testing"

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
