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
}

const (
	defaultConfigPath = "config.yaml"
	defaultLogLevel   = LogLevelInfo
)

const (
	keyConfigFile = "config"
	keyLogLevel   = "log-level"
)

var (
	ErrInvalidLogLevel = errors.New("Invalid log level")
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

	return nil
}
