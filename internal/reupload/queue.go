package reupload

import (
	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/sync/errgroup"
)

type ReuploadQueue struct {
	config    *config.Config
	reuploads *xsync.MapOf[string, reuploadJob]
	uploader  *uploader.Uploader
}

func NewReuploadQueue(config *config.Config, uploader *uploader.Uploader) *ReuploadQueue {
	return &ReuploadQueue{
		config:    config,
		reuploads: xsync.NewMapOf[string, reuploadJob](),
		uploader:  uploader,
	}
}

func (r *ReuploadQueue) Add(path string) {
	job, loaded := r.reuploads.LoadOrStore(path, reuploadJob{path: path, uploader: r.uploader})
	if !loaded {
		go job.Run()
	}
}

func (r *ReuploadQueue) Stop() error {
	errgroup := errgroup.Group{}
	r.reuploads.Range(func(key string, value reuploadJob) bool {
		errgroup.Go(value.Stop)
		return true
	})
	return errgroup.Wait()
}
