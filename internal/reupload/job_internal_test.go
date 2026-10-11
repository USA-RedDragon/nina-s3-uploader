package reupload

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

func TestRunDropsMissingFile(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	root := t.TempDir()
	cfg := &config.Config{
		S3: config.S3{Region: "us-east-1", Bucket: "bucket", Endpoint: "http://127.0.0.1:1"},
		Uploader: config.Uploader{
			Directory: filepath.Join(root, "src"),
			Local:     config.Local{Directory: filepath.Join(root, "local")},
		},
	}
	u, err := uploader.NewUploader(cfg)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "local", "missing.fits")
	job := newReuploadJob(path, u)
	finished := make(chan string, 1)
	go job.Run(func(p string) { finished <- p })

	select {
	case got := <-finished:
		if got != path {
			t.Fatalf("callback got %q, want %q", got, path)
		}
	case <-time.After(5 * time.Second):
		_ = job.Stop()
		t.Fatal("job kept retrying a missing file")
	}
	if job.attempts != 0 {
		t.Fatalf("job scheduled %d retries for a missing file", job.attempts)
	}
}
