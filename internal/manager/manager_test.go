package manager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/manager"
)

func TestNewManagerReturnsUploaderErrors(t *testing.T) {
	awsConfig := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(awsConfig, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_CONFIG_FILE", awsConfig)
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", awsConfig)
	t.Setenv("AWS_PROFILE", "does-not-exist")

	root := t.TempDir()
	m, err := manager.NewManager(&config.Config{
		S3: config.S3{Region: "us-east-1", Bucket: "bucket"},
		Uploader: config.Uploader{
			Directory:  filepath.Join(root, "src"),
			Extensions: []string{".fits"},
			Local:      config.Local{Directory: filepath.Join(root, "local")},
		},
	})
	if err == nil {
		t.Fatalf("expected an error, got manager %v", m)
	}
}
