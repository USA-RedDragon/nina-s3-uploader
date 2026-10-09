package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"syscall"

	"github.com/USA-RedDragon/configulator/v2"
	cpflag "github.com/USA-RedDragon/configulator/v2/flags/pflag"
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
		SilenceErrors:     true,
		DisableAutoGenTag: true,
	}
	loader := config.NewLoader()
	cpflag.Bind(loader, cmd.Flags(), config.ConfigPFlagHooks(), nil)
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runRoot(cmd, loader)
	}
	return cmd
}

func runRoot(cmd *cobra.Command, loader *configulator.Configulator[config.Config]) error {
	fmt.Printf("N.I.N.A S3 Uploader - %s (%s)\n", cmd.Annotations["version"], cmd.Annotations["commit"])

	cfg, err := loader.Load()
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

	if _, err := os.Stat(cfg.Uploader.Directory); os.IsNotExist(err) {
		os.MkdirAll(cfg.Uploader.Directory, os.ModePerm)
	} else if err != nil {
		return fmt.Errorf("failed to check uploader directory: %w", err)
	}

	if _, err := os.Stat(cfg.Uploader.Local.Directory); os.IsNotExist(err) {
		os.MkdirAll(cfg.Uploader.Local.Directory, os.ModePerm)
	} else if err != nil {
		return fmt.Errorf("failed to check local directory: %w", err)
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
