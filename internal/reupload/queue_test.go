package reupload_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/reupload"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

func TestStopStopsRunningJobs(t *testing.T) {
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
	q := reupload.NewQueue(cfg, u)
	q.Add(filepath.Join(root, "local", "missing.fits"))
	time.Sleep(100 * time.Millisecond)

	start := time.Now()
	if err := q.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("stop took %v", elapsed)
	}
}
