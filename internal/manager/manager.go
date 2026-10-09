package manager

import (
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/reupload"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/watcher"
	"github.com/avast/retry-go/v4"
	"golang.org/x/sync/errgroup"
)

type Manager struct {
	config        *config.Config
	srcWatcher    *watcher.Watcher
	localWatcher  *watcher.Watcher
	uploader      *uploader.Uploader
	reuploadQueue *reupload.Queue
}

func NewManager(cfg *config.Config) (*Manager, error) {
	uploader, err := uploader.NewUploader(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create uploader: %w", err)
	}
	localWatcher, err := watcher.NewWatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create local watcher: %w", err)
	}
	watcher, err := watcher.NewWatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create source watcher: %w", err)
	}
	reuploadQueue := reupload.NewQueue(cfg, uploader)

	manager := &Manager{
		config:        cfg,
		srcWatcher:    watcher,
		uploader:      uploader,
		reuploadQueue: reuploadQueue,
		localWatcher:  localWatcher,
	}

	foundFiles := findFiles(cfg.Uploader.Local.Directory, cfg.Uploader.Extensions)
	for _, file := range foundFiles {
		slog.Info("found file in local directory", "path", file)
		go reuploadQueue.Add(file)
	}
	foundFiles = findFiles(cfg.Uploader.Directory, cfg.Uploader.Extensions)
	for _, file := range foundFiles {
		slog.Info("found file in source directory", "path", file)
		go manager.uploadCallback(file)
	}

	return manager, nil
}

func (u *Manager) Start() error {
	u.srcWatcher.SetUploadCallback(u.uploadCallback)
	err := u.srcWatcher.Add(u.config.Uploader.Directory)
	if err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}
	go func() {
		err := u.srcWatcher.Start()
		slog.Debug("source watcher stopped", "error", err)
	}()
	return nil
}

func (u *Manager) Stop() error {
	errgroup := errgroup.Group{}
	errgroup.Go(func() error {
		slog.Debug("stopping source watcher")
		defer slog.Debug("stopped source watcher")
		err := u.srcWatcher.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop source watcher: %w", err)
		}
		return nil
	})
	errgroup.Go(func() error {
		slog.Debug("stopping local watcher")
		defer slog.Debug("stopped local watcher")
		err := u.localWatcher.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop local watcher: %w", err)
		}
		return nil
	})

	errgroup.Go(func() error {
		slog.Debug("stopping reupload queue")
		defer slog.Debug("stopped reupload queue")
		err := u.reuploadQueue.Stop()
		if err != nil {
			return fmt.Errorf("failed to stop reupload queue: %w", err)
		}
		return nil
	})
	return errgroup.Wait()
}

func (u *Manager) uploadCallback(path string) {
	slog.Info("uploading", "path", path)
	err := retry.Do(
		func() error { return u.uploader.Upload(path) },
		retry.Attempts(3),
		retry.DelayType(retry.BackOffDelay),
		retry.Delay(time.Second),
		retry.OnRetry(func(n uint, err error) {
			slog.Warn("retrying upload", "attempt", n+1, "path", path, "error", err)
		}),
	)
	if err != nil {
		slog.Error("failed to upload after 3 attempts", "path", path)

		// path is likely to be an absolute path, but it is not guaranteed to be
		// therefore we should resolve the absolute path in all cases
		path, err := filepath.Abs(path)
		if err != nil {
			slog.Error("failed to resolve absolute path", "path", path, "error", err)
			return
		}

		localPath, err := filepath.Abs(u.config.Uploader.Local.Directory)
		if err != nil {
			slog.Error("failed to resolve absolute path", "path", path, "error", err)
			return
		}

		err = os.MkdirAll(localPath, fs.FileMode(0755))
		if err != nil {
			slog.Error("failed to create local directory", "path", path, "error", err)
			return
		}

		uploaderDirAbsPath, err := filepath.Abs(u.config.Uploader.Directory)
		if err != nil {
			slog.Error("failed to resolve absolute path", "path", path, "error", err)
			return
		}

		// we need to remove the prefix of u.config.Uploader.Directory from the path
		// to be left with only the relative path from the configured local directory
		path, err = filepath.Rel(uploaderDirAbsPath, path)
		if err != nil {
			slog.Error("failed to resolve relative path", "path", path, "error", err)
			return
		}

		localPath = filepath.Join(localPath, path)
		slog.Debug("want to write to local directory", "path", path, "localPath", localPath)

		// Create dir tree in local directory
		err = os.MkdirAll(filepath.Dir(localPath), fs.FileMode(0755))
		if err != nil {
			slog.Error("failed to create local directory", "path", path, "error", err)
			return
		}

		srcFile := filepath.Join(u.config.Uploader.Directory, path)

		// copy file to local directory
		err = copyFile(srcFile, localPath)
		if err != nil {
			slog.Error("failed to copy file to local directory", "path", path, "error", err)
			return
		}
		slog.Info("added to local directory", "path", path)

		if u.config.Uploader.Delay > 0 {
			slog.Debug("delaying for", "delay", u.config.Uploader.Delay)
			time.Sleep(u.config.Uploader.Delay)
			slog.Debug("delay complete")
		}

		err = os.Remove(srcFile)
		if err != nil {
			slog.Error("failed to remove file from source directory", "path", path, "error", err)
			return
		}

		// add an infinite retry to continue trying to upload
		u.reuploadQueue.Add(localPath)
		return
	}
	slog.Info("uploaded", "path", path)

	if u.config.Uploader.Delay > 0 {
		slog.Debug("delaying for", "delay", u.config.Uploader.Delay)
		time.Sleep(u.config.Uploader.Delay)
		slog.Debug("delay complete")
	}

	// File uploaded successfully, remove it
	err = os.Remove(path)
	if err != nil {
		slog.Error("failed to remove file", "path", path, "error", err)
	}
	slog.Info("removed local copy of file", "path", path)
}

func copyFile(src, dst string) error {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	_, err = io.Copy(destination, source)
	if err == nil {
		err = destination.Sync()
	}
	if closeErr := destination.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		removeErr := os.Remove(dst)
		if removeErr != nil {
			slog.Error("failed to cleanup failed copy", "path", dst, "error", removeErr)
		}
		return err
	}
	return nil
}

func findFiles(path string, extensions []string) []string {
	var files []string
	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil && !os.IsPermission(err) {
			return err
		} else if os.IsPermission(err) {
			return filepath.SkipDir
		}
		if watcher.IsBadDir(path) {
			return filepath.SkipDir
		}
		if !info.IsDir() && slices.Contains(extensions, filepath.Ext(path)) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		slog.Error("failed to walk directory", "path", path, "error", err)
	}
	return files
}
