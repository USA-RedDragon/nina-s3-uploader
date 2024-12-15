package manager

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
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
	reuploadQueue *reupload.ReuploadQueue
}

func NewManager(cfg *config.Config) (*Manager, error) {
	localWatcher, err := watcher.NewWatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create local watcher: %w", err)
	}
	watcher, err := watcher.NewWatcher(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create source watcher: %w", err)
	}
	// TODO: walk the local directory and startup reupload jobs for each file
	// TODO: walk the source directory and upload each file
	uploader, err := uploader.NewUploader(cfg)
	return &Manager{
		config:        cfg,
		srcWatcher:    watcher,
		uploader:      uploader,
		localWatcher:  localWatcher,
		reuploadQueue: reupload.NewReuploadQueue(cfg, uploader),
	}, nil
}

func (u *Manager) Start() error {
	u.srcWatcher.SetUploadCallback(u.uploadCallback)
	err := u.srcWatcher.Add(u.config.Uploader.Directory)
	if err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}
	go u.srcWatcher.Start()
	return nil
}

func (u *Manager) Stop() error {
	errgroup := errgroup.Group{}
	errgroup.Go(u.srcWatcher.Stop)
	errgroup.Go(u.localWatcher.Stop)
	errgroup.Go(u.reuploadQueue.Stop)
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
		os.MkdirAll(filepath.Dir(localPath), fs.FileMode(0755))

		srcFile := filepath.Join(u.config.Uploader.Directory, path)

		// copy file to local directory
		err = copyFile(srcFile, localPath)
		if err != nil {
			slog.Error("failed to copy file to local directory", "path", path, "error", err)
			return
		}
		slog.Info("added to local directory", "path", path)

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

	// Check if the destination file exists
	// and if it does, remove it
	if _, err := os.Stat(dst); errors.Is(err, os.ErrNotExist) {
		err = os.Remove(dst)
		if err != nil {
			slog.Error("failed to remove existing file", "path", dst, "error", err)
		}
	}

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	buf := make([]byte, 1024)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			removeErr := os.Remove(dst)
			if removeErr != nil {
				slog.Error("failed to cleanup failed copy", "path", dst, "error", removeErr)
			}
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			removeErr := os.Remove(dst)
			if removeErr != nil {
				slog.Error("failed to cleanup failed copy", "path", dst, "error", removeErr)
			}
			return err
		}
	}
	return nil
}
