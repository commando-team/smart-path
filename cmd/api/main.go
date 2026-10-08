package main

import (
	"fmt"
	"log/slog"

	"github.com/commando-team/smart-path/internal/config"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.MustLoad()

	log := logger.New(cfg)

	switch cfg.Environment {
	case config.EnvDevelopment:
		gin.SetMode(gin.DebugMode)
	case config.EnvRelease:
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	if err := r.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Error("failed running the server", slog.Any("err", err))
	}
}
