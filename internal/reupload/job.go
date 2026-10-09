package reupload

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"os"
	"sync"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

type reuploadJob struct {
	path     string
	uploader *uploader.Uploader
	attempts uint64
	stop     chan struct{}
	stopOnce sync.Once
	done     chan struct{}
}

func newReuploadJob(path string, uploader *uploader.Uploader) *reuploadJob {
	return &reuploadJob{
		path:     path,
		uploader: uploader,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

func (r *reuploadJob) Run(callback func(path string)) {
	defer func() {
		close(r.done)
		callback(r.path)
	}()
	for {
		select {
		case <-r.stop:
			return
		default:
		}
		err := r.uploader.Upload(r.path)
		if err == nil {
			break
		}
		slog.Warn("retrying upload", "attempt", r.attempts+1, "path", r.path, "error", err)
		r.attempts++
		randomJitter := rand.N(5 * time.Minute).Truncate(time.Second) //nolint:gosec // retry jitter, not security sensitive
		slog.Debug("sleeping before retrying", "duration", randomJitter)
		select {
		case <-r.stop:
			return
		case <-time.After(randomJitter):
		}
	}
	err := os.Remove(r.path)
	if err != nil {
		slog.Error("failed to remove file from local directory", "path", r.path, "error", err)
	}
}

func (r *reuploadJob) Stop() error {
	r.stopOnce.Do(func() { close(r.stop) })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	select {
	case <-r.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
