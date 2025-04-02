package watcher

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/fsnotify/fsnotify"
	"github.com/puzpuzpuz/xsync/v4"
)

type Watcher struct {
	config    *config.Config
	fsWatcher *fsnotify.Watcher
	callback  UploadCallback
	debounces *xsync.MapOf[string, time.Time]
}

var BadDirs = []*regexp.Regexp{
	regexp.MustCompile("System Volume Information(\\.*)?"),
	regexp.MustCompile("lost+found(/.*)?"),
	regexp.MustCompile("\\$RECYCLE.BIN(\\.*)?"),
}

type UploadCallback func(path string)

func NewWatcher(cfg *config.Config) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	return &Watcher{
		config:    cfg,
		fsWatcher: watcher,
		debounces: xsync.NewMapOf[string, time.Time](),
	}, nil
}

func (u *Watcher) SetUploadCallback(callback UploadCallback) {
	u.callback = callback
}

func (u *Watcher) Start() error {
	for {
		select {
		case event, ok := <-u.fsWatcher.Events:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			slog.Debug("event", "event", event)
			go u.processEvent(event)
		case err, ok := <-u.fsWatcher.Errors:
			if !ok {
				return fmt.Errorf("watcher channel closed")
			}
			slog.Error("error", "error", err)
		}
	}
}

func (u *Watcher) Stop() error {
	return u.fsWatcher.Close()
}

func (u *Watcher) Add(path string) error {
	dirs, err := walkdir(u.config.Uploader.Directory)
	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}
	slog.Debug("walked", "dirs", dirs)
	for _, dir := range dirs {
		if !slices.Contains(u.fsWatcher.WatchList(), dir) {
			err := u.fsWatcher.Add(dir)
			slog.Info("watching", "path", dir)
			if err != nil {
				return fmt.Errorf("failed to add directory to watcher: %w", err)
			}
		}
	}

	return nil
}

func (u *Watcher) debounce(path string, callback UploadCallback) {
	t, ok := u.debounces.LoadAndStore(path, time.Now())
	if ok {
		slog.Debug("debouncing", "path", path)
		return
	}
	go func() {
		for {
			if time.Since(t) > 5*time.Second {
				u.debounces.Delete(path)
				callback(path)
				break
			}
			time.Sleep(1 * time.Second)
			t, ok = u.debounces.Load(path)
			if !ok {
				break
			}
		}
	}()
}

func (u *Watcher) processEvent(event fsnotify.Event) {
	switch event.Op {
	case fsnotify.Create:
		fstat, err := os.Stat(event.Name)
		if err != nil {
			slog.Error("failed to stat", "path", event.Name, "error", err)
			return
		}
		if fstat.IsDir() {
			u.Add(event.Name)
		} else {
			// This is probably a new file, so we should upload it
			if slices.Contains(u.config.Uploader.Extensions, filepath.Ext(event.Name)) {
				slog.Info("new file", "path", event.Name)
			}
		}
	case fsnotify.Write:
		slog.Info("modified", "path", event.Name)
		if slices.Contains(u.config.Uploader.Extensions, filepath.Ext(event.Name)) {
			slog.Info("wrote file", "path", event.Name)
			u.debounce(event.Name, u.callback)
		}
	case fsnotify.Remove:
		// no-op, the watch list is automatically updated when a file is renamed
	case fsnotify.Rename:
		// no-op, the watch list is automatically updated when a file is renamed
	case fsnotify.Chmod:
		// no-op, we don't care about file permissions
	}
}

func walkdir(dir string) ([]string, error) {
	var dirs []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if d.IsDir() {
			if err != nil && !os.IsPermission(err) {
				return err
			} else if os.IsPermission(err) {
				return filepath.SkipDir
			}
			for _, badDir := range BadDirs {
				if badDir.MatchString(d.Name()) {
					return filepath.SkipDir
				}
			}
			dirs = append(dirs, path)
		}
		return nil
	})
	return dirs, err
}
