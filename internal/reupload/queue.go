package reupload

import (
	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
	"github.com/puzpuzpuz/xsync/v3"
	"golang.org/x/sync/errgroup"
)

type Queue struct {
	config    *config.Config
	reuploads *xsync.MapOf[string, *reuploadJob]
	uploader  *uploader.Uploader
}

func NewQueue(config *config.Config, uploader *uploader.Uploader) *Queue {
	return &Queue{
		config:    config,
		reuploads: xsync.NewMapOf[string, *reuploadJob](),
		uploader:  uploader,
	}
}

func (r *Queue) Add(path string) {
	job, loaded := r.reuploads.LoadOrStore(path, newReuploadJob(path, r.uploader))
	if !loaded {
		go job.Run(r.callback)
	}
}

func (r *Queue) callback(path string) {
	r.reuploads.Delete(path)
}

func (r *Queue) Stop() error {
	errgroup := errgroup.Group{}
	r.reuploads.Range(func(_ string, value *reuploadJob) bool {
		errgroup.Go(value.Stop)
		return true
	})
	return errgroup.Wait()
}
