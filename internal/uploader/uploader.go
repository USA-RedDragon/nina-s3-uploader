package uploader

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Uploader struct {
	config    *config.Config
	s3Client  *s3.Client
	s3Manager *manager.Uploader
	upload    *uploadJob
	lock      sync.Mutex
}

func NewUploader(cfg *config.Config) (*Uploader, error) {
	awsCfg, err := awsConfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	ret := &Uploader{
		config: cfg,
		s3Client: s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.Region = cfg.S3.Region
			if cfg.S3.Endpoint != "" {
				slog.Warn("using custom S3 endpoint", "endpoint", cfg.S3.Endpoint)
				o.BaseEndpoint = aws.String(cfg.S3.Endpoint)
				o.UsePathStyle = true
			}
		}),
	}

	ret.s3Manager = manager.NewUploader(ret.s3Client, func(o *manager.Uploader) {
		o.PartSize = 16 * 1024 * 1024 // 16MB
		o.LeavePartsOnError = false
		o.Concurrency = 5
		o.ClientOptions = []func(*s3.Options){
			func(c *s3.Options) {
				c.Region = cfg.S3.Region
				if cfg.S3.Endpoint != "" {
					slog.Warn("using custom S3 endpoint", "endpoint", cfg.S3.Endpoint)
					c.BaseEndpoint = aws.String(cfg.S3.Endpoint)
					c.UsePathStyle = true
				}
			},
		}
	})

	return ret, nil
}

func (u *Uploader) Upload(path string) error {
	u.lock.Lock()
	defer u.lock.Unlock()

	if u.upload == nil {
		u.upload = &uploadJob{
			path:      path,
			s3Client:  u.s3Client,
			config:    u.config,
			s3Manager: u.s3Manager,
		}
		err := u.upload.Run()
		u.upload = nil
		if err != nil {
			return fmt.Errorf("failed to upload file: %w", err)
		}
		return nil
	} else {
		return fmt.Errorf("upload already in progress")
	}
}
