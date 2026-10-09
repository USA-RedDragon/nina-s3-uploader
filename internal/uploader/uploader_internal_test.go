package uploader

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type recordingTransport struct {
	url string
}

var errRecorded = errors.New("recorded")

func (r *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.url = req.URL.String()
	return nil, errRecorded
}

func TestDefaultEndpointReachesAWS(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")

	cfg, err := config.NewLoader().Default()
	if err != nil {
		t.Fatal(err)
	}
	cfg.S3.Bucket = "bucket"
	u, err := NewUploader(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	transport := &recordingTransport{}
	_, err = u.s3Client.HeadObject(context.Background(),
		&s3.HeadObjectInput{Bucket: aws.String("bucket"), Key: aws.String("key")},
		func(o *s3.Options) {
			o.HTTPClient = &http.Client{Transport: transport}
			o.RetryMaxAttempts = 1
		})
	if !errors.Is(err, errRecorded) {
		t.Fatalf("request was not sent: %v", err)
	}
	if transport.url != "https://bucket.s3.us-east-1.amazonaws.com/key" {
		t.Errorf("request went to %s", transport.url)
	}
}
