package cmd

import (
	"fmt"
	"log/slog"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/spf13/cobra"
)

func NewCommand(version, commit string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "nina-s3-uploader",
		Version: fmt.Sprintf("%s - %s", version, commit),
		Annotations: map[string]string{
			"version": version,
			"commit":  commit,
		},
		RunE:              runRoot,
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
	config.RegisterFlags(cmd)
	return cmd
}

func runRoot(cmd *cobra.Command, _ []string) error {
	slog.Info("N.I.N.A S3 Uploader", "version", cmd.Annotations["version"], "commit", cmd.Annotations["commit"])

	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	switch cfg.LogLevel {
	case config.LogLevelDebug:
		slog.SetLogLoggerLevel(slog.LevelDebug)
	case config.LogLevelInfo:
		slog.SetLogLoggerLevel(slog.LevelInfo)
	case config.LogLevelWarn:
		slog.SetLogLoggerLevel(slog.LevelWarn)
	case config.LogLevelError:
		slog.SetLogLoggerLevel(slog.LevelError)
	}

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	return nil
}
