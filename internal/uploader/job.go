package uploader

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
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
	rel, ok := relativeTo(u.config.Uploader.Local.Directory, u.path)
	if !ok {
		rel, ok = relativeTo(u.config.Uploader.Directory, u.path)
	}
	if !ok {
		return fmt.Errorf("%s is not in the local or source directory", u.path)
	}
	u.path = strings.ReplaceAll(rel, "\\", "/")
	key := strings.TrimPrefix(path.Join(u.config.S3.Prefix, u.path), "/")

	slog.Debug("uploading file", "path", u.path, "bucket", u.config.S3.Bucket, "prefix", u.config.S3.Prefix)
	_, err = u.s3Manager.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.S3.Bucket),
		Key:    aws.String(key),
		Body:   file,
	})
	if err != nil {
		slog.Error("failed to upload file", "path", u.path, "error", err)
		return err
	} else {
		err = s3.NewObjectExistsWaiter(u.s3Client).Wait(
			context.TODO(), &s3.HeadObjectInput{Bucket: aws.String(u.config.S3.Bucket), Key: aws.String(key)}, time.Minute)
		if err != nil {
			slog.Error("failed to wait for object to exist", "path", u.path, "error", err)
			return err
		}
	}
	slog.Debug("uploaded file", "path", u.path, "bucket", u.config.S3.Bucket, "prefix", u.config.S3.Prefix)
	return nil
}

// relativeTo returns path relative to dir, and false if path is not inside dir.
func relativeTo(dir, path string) (string, bool) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", false
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(absDir, absPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return rel, true
}
