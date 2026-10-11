package manager

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/reupload"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/uploader"
)

func TestUploadCallbackDropsMissingFile(t *testing.T) {
	t.Setenv("AWS_ACCESS_KEY_ID", "id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "secret")
	t.Setenv("AWS_CONFIG_FILE", "/nonexistent")
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/nonexistent")
	root := t.TempDir()
	cfg := &config.Config{
		S3: config.S3{Region: "us-east-1", Bucket: "bucket", Endpoint: "http://127.0.0.1:1"},
		Uploader: config.Uploader{
			Directory:  filepath.Join(root, "src"),
			Extensions: []string{".fits"},
			Local:      config.Local{Directory: filepath.Join(root, "local")},
		},
	}
	u, err := uploader.NewUploader(cfg)
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{config: cfg, uploader: u, reuploadQueue: reupload.NewQueue(cfg, u)}

	start := time.Now()
	m.uploadCallback(filepath.Join(root, "src", "night", "missing.fits"))
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("missing file was retried for %v", elapsed)
	}
	if _, err := os.Stat(filepath.Join(root, "local")); !os.IsNotExist(err) {
		t.Fatalf("local directory was created for a missing file: %v", err)
	}
}
