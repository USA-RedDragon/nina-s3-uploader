package uploader

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type uploadJob struct {
	path      string
	s3Client  *s3.Client
	s3Manager *manager.Uploader
	config    *config.Config
}

func (u *uploadJob) Run() error {
	file, err := os.Open(u.path)
	if err != nil {
		slog.Error("failed to open file", "path", u.path, "error", err)
		return err
	}
	defer file.Close()
	if strings.HasPrefix(u.path, u.config.Uploader.Local.Directory) {
		u.path = strings.TrimPrefix(u.path, u.config.Uploader.Local.Directory)
	} else if strings.HasPrefix(u.path, u.config.Uploader.Directory) {
		u.path = strings.TrimPrefix(u.path, u.config.Uploader.Directory)
	} else {
		slog.Error("file path does not match local or source directory", "path", u.path)
		return nil
	}

	u.path = strings.ReplaceAll(u.path, "\\", "/")

	slog.Debug("uploading file", "path", u.path, "bucket", u.config.S3.Bucket, "prefix", u.config.S3.Prefix)
	_, err = u.s3Manager.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.S3.Bucket),
		Key:    aws.String(u.config.S3.Prefix + u.path),
		Body:   file,
	})
	if err != nil {
		slog.Error("failed to upload file", "path", u.path, "error", err)
		return err
	} else {
		err = s3.NewObjectExistsWaiter(u.s3Client).Wait(
			context.TODO(), &s3.HeadObjectInput{Bucket: aws.String(u.config.S3.Bucket), Key: aws.String(u.config.S3.Prefix + u.path)}, time.Minute)
		if err != nil {
			slog.Error("failed to wait for object to exist", "path", u.path, "error", err)
			return err
		}
	}
	slog.Debug("uploaded file", "path", u.path, "bucket", u.config.S3.Bucket, "prefix", u.config.S3.Prefix)
	return nil
}
