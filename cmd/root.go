package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/USA-RedDragon/nina-s3-uploader/internal/config"
	"github.com/USA-RedDragon/nina-s3-uploader/internal/manager"
	"github.com/lmittmann/tint"
	"github.com/spf13/cobra"
	"github.com/ztrue/shutdown"
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
	fmt.Printf("N.I.N.A S3 Uploader - %s (%s)\n", cmd.Annotations["version"], cmd.Annotations["commit"])

	cfg, err := config.LoadConfig(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	var logger *slog.Logger
	switch cfg.LogLevel {
	case config.LogLevelDebug:
		logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelDebug}))
	case config.LogLevelInfo:
		logger = slog.New(tint.NewHandler(os.Stdout, &tint.Options{Level: slog.LevelInfo}))
	case config.LogLevelWarn:
		logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: slog.LevelWarn}))
	case config.LogLevelError:
		logger = slog.New(tint.NewHandler(os.Stderr, &tint.Options{Level: slog.LevelError}))
	}
	slog.SetDefault(logger)

	err = cfg.Validate()
	if err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}

	manager, err := manager.NewManager(cfg)
	if err != nil {
		return fmt.Errorf("failed to create manager: %w", err)
	}
	err = manager.Start()
	if err != nil {
		return fmt.Errorf("failed to start manager: %w", err)
	}

	stop := func(_ os.Signal) {
		// Skip a line so the control characters don't mess up the output
		fmt.Println("")
		slog.Info("Shutting down")

		err := manager.Stop()
		if err != nil {
			slog.Error("Shutdown error", "error", err.Error())
		}
		slog.Info("Shutdown complete")
	}
	shutdown.AddWithParam(stop)
	shutdown.Listen(syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGQUIT)

	return nil
}
