package logger

import (
	"log/slog"
	"os"

	"github.com/commando-team/smart-path/internal/config"
)

func New(cfg *config.Config) *slog.Logger {
	var log *slog.Logger

	switch cfg.Environment {
	case config.EnvDevelopment:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case config.EnvRelease:
		slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}
