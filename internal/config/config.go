package config

import (
	"errors"
	"time"

	"github.com/USA-RedDragon/configulator/v2"
	"github.com/goccy/go-yaml"
)

//go:generate go tool configulator -type Config

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Config stores the application configuration.
type Config struct {
	LogLevel LogLevel `name:"log-level" default:"info" description:"Log level, one of debug, info, warn or error"`

	S3       S3       `name:"s3"`
	Uploader Uploader `name:"uploader"`
}

// S3 configures the bucket files are uploaded to.
type S3 struct {
	Region   string `name:"region" default:"us-east-1" description:"The region to use"`
	Bucket   string `name:"bucket" required:"true" description:"The bucket to upload to"`
	Prefix   string `name:"prefix" default:"/" description:"The prefix to use for the uploaded files"`
	Endpoint string `name:"endpoint" description:"A custom S3-compatible endpoint URL, such as https://minio.example.com. Empty uses AWS"`
}

// Uploader configures which files are uploaded and where failed uploads are kept.
type Uploader struct {
	Directory  string        `name:"directory" required:"true" description:"The directory to watch for new files"`
	Extensions []string      `name:"extensions" required:"true" description:"The file extensions to watch for, such as .fits. Comma-separated in an environment variable"`
	Local      Local         `name:"local"`
	Delay      time.Duration `name:"delay" description:"How long to wait after a file is uploaded or moved to the local directory before removing it from the watched directory"`
}

// Local configures where files are kept when they fail to upload.
type Local struct {
	Directory string `name:"directory" required:"true" description:"Files are only stored here if they fail to upload to S3. Once a file uploads at a later time, it is deleted from this directory"`
}

var ErrInvalidLogLevel = errors.New("Invalid log level")

// NewLoader returns a configulator that reads config.yaml if it exists and
// environment variables such as S3__BUCKET. Bind flags to it before loading.
func NewLoader() *configulator.Configulator[Config] {
	return configulator.New(ConfigSchema()).
		WithEnvironmentVariables(&configulator.EnvironmentVariableOptions{Separator: "__"}).
		WithFile(&configulator.FileOptions{
			Search:   []string{"config.yaml"},
			Decoders: configulator.Decoders{".yaml": yaml.Unmarshal, ".yml": yaml.Unmarshal},
		})
}

func (c Config) Validate() error {
	switch c.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}

	return nil
}
