package config_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/USA-RedDragon/configulator/v2"
	cpflag "github.com/USA-RedDragon/configulator/v2/flags/pflag"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/spf13/pflag"
)

func load(t *testing.T, env map[string]string, args ...string) (*config.Config, error) {
	t.Helper()
	loader := config.NewLoader().WithEnviron(func(k string) (string, bool) {
		v, ok := env[k]
		return v, ok
	})
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	cpflag.Bind(loader, fs, config.ConfigPFlagHooks(), nil)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	return loader.Load()
}

func want(t *testing.T, cfg *config.Config) {
	t.Helper()
	if cfg.LogLevel != config.LogLevelDebug {
		t.Errorf("log-level = %q", cfg.LogLevel)
	}
	if cfg.S3.Region != "eu-west-1" {
		t.Errorf("s3.region = %q", cfg.S3.Region)
	}
	if cfg.S3.Bucket != "bucket" {
		t.Errorf("s3.bucket = %q", cfg.S3.Bucket)
	}
	if cfg.S3.Prefix != "astro/" {
		t.Errorf("s3.prefix = %q", cfg.S3.Prefix)
	}
	if cfg.S3.Endpoint != "http://minio:9000" {
		t.Errorf("s3.endpoint = %q", cfg.S3.Endpoint)
	}
	if cfg.Uploader.Directory != "/src" {
		t.Errorf("uploader.directory = %q", cfg.Uploader.Directory)
	}
	if !slices.Equal(cfg.Uploader.Extensions, []string{".fits", ".xisf"}) {
		t.Errorf("uploader.extensions = %q", cfg.Uploader.Extensions)
	}
	if cfg.Uploader.Delay != 90*time.Second {
		t.Errorf("uploader.delay = %v", cfg.Uploader.Delay)
	}
	if cfg.Uploader.Local.Directory != "/local" {
		t.Errorf("uploader.local.directory = %q", cfg.Uploader.Local.Directory)
	}
}

func TestEveryFieldFromEnv(t *testing.T) {
	t.Parallel()
	cfg, err := load(t, map[string]string{
		"LOG_LEVEL":                  "debug",
		"S3__REGION":                 "eu-west-1",
		"S3__BUCKET":                 "bucket",
		"S3__PREFIX":                 "astro/",
		"S3__ENDPOINT":               "http://minio:9000",
		"UPLOADER__DIRECTORY":        "/src",
		"UPLOADER__EXTENSIONS":       ".fits,.xisf",
		"UPLOADER__DELAY":            "1m30s",
		"UPLOADER__LOCAL__DIRECTORY": "/local",
	})
	if err != nil {
		t.Fatal(err)
	}
	want(t, cfg)
}

func TestEveryFieldFromFlags(t *testing.T) {
	t.Parallel()
	cfg, err := load(t, nil,
		"--log-level", "debug",
		"--s3.region", "eu-west-1",
		"--s3.bucket", "bucket",
		"--s3.prefix", "astro/",
		"--s3.endpoint", "http://minio:9000",
		"--uploader.directory", "/src",
		"--uploader.extensions", ".fits",
		"--uploader.extensions", ".xisf",
		"--uploader.delay", "1m30s",
		"--uploader.local.directory", "/local",
	)
	if err != nil {
		t.Fatal(err)
	}
	want(t, cfg)
}

func TestFlagsOverrideEnv(t *testing.T) {
	t.Parallel()
	cfg, err := load(t, map[string]string{
		"S3__BUCKET":                 "env",
		"UPLOADER__DIRECTORY":        "/src",
		"UPLOADER__EXTENSIONS":       ".fits",
		"UPLOADER__LOCAL__DIRECTORY": "/local",
	}, "--s3.bucket", "flag")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.Bucket != "flag" {
		t.Errorf("s3.bucket = %q", cfg.S3.Bucket)
	}
}

func TestExistingConfigFileLoads(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	err := os.WriteFile(path, []byte(`log-level: debug
s3:
  region: eu-west-1
  bucket: bucket
  prefix: astro/
  endpoint: http://minio:9000
uploader:
  directory: /src
  extensions:
    - .fits
    - .xisf
  delay: 1m30s
  local:
    directory: /local
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := load(t, nil, "--config", path)
	if err != nil {
		t.Fatal(err)
	}
	want(t, cfg)
}

func TestDefaults(t *testing.T) {
	t.Parallel()
	cfg, err := load(t, map[string]string{
		"S3__BUCKET":                 "bucket",
		"UPLOADER__DIRECTORY":        "/src",
		"UPLOADER__EXTENSIONS":       ".fits",
		"UPLOADER__LOCAL__DIRECTORY": "/local",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.LogLevel != config.LogLevelInfo || cfg.S3.Region != "us-east-1" || cfg.S3.Prefix != "/" || cfg.Uploader.Delay != 0 {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestMissingExplicitConfigFile(t *testing.T) {
	t.Parallel()
	_, err := load(t, nil, "--config", filepath.Join(t.TempDir(), "missing.yaml"))
	var missing *configulator.MissingFileError
	if !errors.As(err, &missing) {
		t.Fatalf("expected a MissingFileError, got %v", err)
	}
}

func TestRequiredFields(t *testing.T) {
	t.Parallel()
	full := map[string]string{
		"s3.bucket":                "S3__BUCKET",
		"uploader.directory":       "UPLOADER__DIRECTORY",
		"uploader.extensions":      "UPLOADER__EXTENSIONS",
		"uploader.local.directory": "UPLOADER__LOCAL__DIRECTORY",
	}
	for key := range full {
		t.Run(key, func(t *testing.T) {
			t.Parallel()
			env := map[string]string{}
			for k, name := range full {
				if k != key {
					env[name] = "x"
				}
			}
			_, err := load(t, env)
			var required *configulator.RequiredError
			if !errors.As(err, &required) || required.Path != key {
				t.Fatalf("expected %s to be required, got %v", key, err)
			}
		})
	}
}

func TestInvalidLogLevel(t *testing.T) {
	t.Parallel()
	_, err := load(t, map[string]string{
		"LOG_LEVEL":                  "verbose",
		"S3__BUCKET":                 "bucket",
		"UPLOADER__DIRECTORY":        "/src",
		"UPLOADER__EXTENSIONS":       ".fits",
		"UPLOADER__LOCAL__DIRECTORY": "/local",
	})
	if !errors.Is(err, config.ErrInvalidLogLevel) {
		t.Fatalf("expected ErrInvalidLogLevel, got %v", err)
	}
}

func TestOldExampleConfigLoadsFromSearchPath(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(`# Log configuration, one of debug, info, warn, error
log-level: info

s3:
  region: us-east-1
  bucket: YOUR_BUCKET_NAME
  prefix: /
  endpoint: https://s3.amazonaws.com

uploader:
  directory: R:\
  extensions:
    - .fits
  local:
    directory: C:\Users\your\directory
`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	cfg, err := load(t, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.S3.Bucket != "YOUR_BUCKET_NAME" || cfg.Uploader.Directory != `R:\` ||
		cfg.Uploader.Local.Directory != `C:\Users\your\directory` || !slices.Equal(cfg.Uploader.Extensions, []string{".fits"}) {
		t.Errorf("unexpected config: %+v", cfg)
	}
}
