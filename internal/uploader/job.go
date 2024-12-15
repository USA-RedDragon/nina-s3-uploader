package uploader

import (
	"bufio"
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type uploadJob struct {
	path     string
	s3Client *s3.Client
	config   *config.Config
}

func (u *uploadJob) Run() error {
	file, err := os.Open(u.path)
	if err != nil {
		slog.Error("failed to open file", "path", u.path, "error", err)
		return err
	}
	defer file.Close()
	slog.Debug("uploading file", "path", u.path, "bucket", u.config.S3.Bucket, "prefix", u.config.S3.Prefix)
	_, err = u.s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(u.config.S3.Bucket),
		Key:    aws.String(u.config.S3.Prefix + u.path),
		Body:   bufio.NewReader(file),
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
	return nil
}
