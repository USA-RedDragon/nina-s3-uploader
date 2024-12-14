package uploader

import (
	"fmt"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
)

type Uploader struct {
	config *config.Config
}

func NewUploader(cfg *config.Config) *Uploader {
	return &Uploader{
		config: cfg,
	}
}

func (u *Uploader) Upload(path string) error {
	return fmt.Errorf("not implemented")
}
