package reupload

import (
	"context"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

type reuploadJob struct {
	path     string
	started  bool
	stopped  bool
	uploader *uploader.Uploader
	attempts uint64
}

func (r *reuploadJob) Run() error {
	r.started = true
	defer func() { r.stopped = true }()
	for r.started {
		err := r.uploader.Upload(r.path)
		if err != nil {
			slog.Warn("retrying upload", "attempt", r.attempts+1, "path", r.path, "error", err)
			slog.Error("failed to upload file", "path", r.path, "error", err)
			r.attempts++
			randomJitter := time.Duration(rand.Intn(5))*time.Minute + time.Duration(rand.Intn(60))*time.Second
			slog.Debug("sleeping before retrying", "duration", randomJitter)
			time.Sleep(randomJitter)
			continue
		}
		err = os.Remove(r.path)
		if err != nil {
			slog.Error("failed to remove file from local directory", "path", r.path, "error", err)
			return err
		}

		return nil
	}
	return nil
}

func (r *reuploadJob) Stop() error {
	r.started = false

	var done = make(chan struct{})
	go func() {
		for !r.stopped {
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(10*time.Second))
	defer cancel()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
