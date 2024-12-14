package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// Config stores the application configuration.
type Config struct {
	LogLevel LogLevel `json:"log-level" yaml:"log-level"`

	S3       S3       `json:"s3" yaml:"s3"`
	Uploader Uploader `json:"uploader" yaml:"uploader"`
}

type S3 struct {
	Region          string `json:"region" yaml:"region"`
	AccessKeyID     string `json:"access-key-id" yaml:"access-key-id"`
	SecretAccessKey string `json:"secret-access-key" yaml:"secret-access-key"`
	Bucket          string `json:"bucket" yaml:"bucket"`
	Prefix          string `json:"prefix" yaml:"prefix"`
	Endpoint        string `json:"endpoint" yaml:"endpoint"`
}

type Uploader struct {
	Directory  string   `json:"directory" yaml:"directory"`
	Extensions []string `json:"extensions" yaml:"extensions"`
	Local      Local    `json:"local" yaml:"local"`
}

type Local struct {
	Directory string `json:"directory" yaml:"directory"`
}

const (
	defaultConfigPath = "config.yaml"
	defaultLogLevel   = LogLevelInfo

	defaultS3Region   = "us-east-1"
	defaultS3Prefix   = "/"
	defaultS3Endpoint = "s3.amazonaws.com"
)

const (
	keyConfigFile = "config"
	keyLogLevel   = "log-level"

	keyS3Region          = "s3.region"
	keyS3AccessKeyID     = "s3.access-key-id"
	keyS3SecretAccessKey = "s3.secret-access-key"
	keyS3Bucket          = "s3.bucket"
	keyS3Prefix          = "s3.prefix"
	keyS3Endpoint        = "s3.endpoint"

	keyUploaderDirectory      = "uploader.directory"
	keyUploaderExtensions     = "uploader.extensions"
	keyUploaderLocalDirectory = "uploader.local.directory"
)

var (
	ErrInvalidLogLevel           = errors.New("Invalid log level")
	ErrMissingAWSAccessKeyID     = errors.New("Missing AWS access key ID")
	ErrMissingAWSSecretAccessKey = errors.New("Missing AWS secret access key")
	ErrMissingS3Bucket           = errors.New("Missing S3 bucket")
	ErrMissingUploaderDirectory  = errors.New("Missing uploader directory")
	ErrMissingUploaderExtensions = errors.New("Missing uploader extensions")
	ErrMissingUploaderLocalDir   = errors.New("Missing uploader local directory")
)

func LoadConfig(cmd *cobra.Command) (*Config, error) {
	var config Config

	// Load flags from envs
	ctx, cancel := context.WithCancelCause(cmd.Context())
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if ctx.Err() != nil {
			return
		}
		optName := strings.ReplaceAll(strings.ReplaceAll(strings.ToUpper(f.Name), "-", "_"), ".", "__")
		if val, ok := os.LookupEnv(optName); !f.Changed && ok {
			if err := f.Value.Set(val); err != nil {
				cancel(err)
			}
			f.Changed = true
		}
	})
	if ctx.Err() != nil {
		return &config, fmt.Errorf("failed to load env: %w", context.Cause(ctx))
	}

	configPath, err := cmd.Flags().GetString("config")
	if err != nil {
		return &config, fmt.Errorf("failed to get config path: %w", err)
	}
	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return &config, fmt.Errorf("failed to read config: %w", err)
		} else if err == nil {
			if err := yaml.Unmarshal(data, &config); err != nil {
				return &config, fmt.Errorf("failed to unmarshal config: %w", err)
			}
		}
	}

	err = overrideFlags(&config, cmd)
	if err != nil {
		return &config, fmt.Errorf("failed to override flags: %w", err)
	}

	// Defaults
	if config.LogLevel == "" {
		config.LogLevel = defaultLogLevel
	}
	if config.S3.Region == "" {
		config.S3.Region = defaultS3Region
	}
	if config.S3.Prefix == "" {
		config.S3.Prefix = defaultS3Prefix
	}
	if config.S3.Endpoint == "" {
		config.S3.Endpoint = defaultS3Endpoint
	}

	return &config, nil
}

func RegisterFlags(cmd *cobra.Command) {
	cmd.Flags().StringP(keyConfigFile, "c", defaultConfigPath, "Config file path")
	cmd.Flags().String(keyLogLevel, string(defaultLogLevel), "Log level")
}

func overrideFlags(config *Config, cmd *cobra.Command) error {
	if cmd.Flags().Changed(keyLogLevel) {
		ll, err := cmd.Flags().GetString(keyLogLevel)
		if err != nil {
			return fmt.Errorf("failed to get log level: %w", err)
		}
		config.LogLevel = LogLevel(ll)
	}

	return nil
}

func (c *Config) Validate() error {
	switch c.LogLevel {
	case LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError:
	default:
		return ErrInvalidLogLevel
	}
	if c.S3.AccessKeyID == "" {
		return ErrMissingAWSAccessKeyID
	}
	if c.S3.SecretAccessKey == "" {
		return ErrMissingAWSSecretAccessKey
	}
	if c.S3.Bucket == "" {
		return ErrMissingS3Bucket
	}
	if c.Uploader.Directory == "" {
		return ErrMissingUploaderDirectory
	}
	if len(c.Uploader.Extensions) == 0 {
		return ErrMissingUploaderExtensions
	}
	if c.Uploader.Local.Directory == "" {
		return ErrMissingUploaderLocalDir
	}

	return nil
}
