package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/commando-team/smart-path/internal/config"
	"github.com/commando-team/smart-path/internal/http/handler"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/commando-team/smart-path/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	cfg := config.MustLoad()

	switch cfg.Environment {
	case config.EnvDevelopment:
		gin.SetMode(gin.DebugMode)
	case config.EnvRelease:
		gin.SetMode(gin.ReleaseMode)
	}

	log := logger.New(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// строка подключения к бд
	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	// накатываем миграции
	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		log.Error("failed to initialize migration", logger.Err(err))
		os.Exit(1)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.Error("migration error", logger.Err(err))
		os.Exit(1)
	}
	m.Close()

	// пул соединений к бд
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("db connection error", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()

	paymentService := service.NewPaymentService()

	apiHandler := handler.NewApiHandler(log, paymentService)

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.POST("/payment", apiHandler.Pay)

	if err := r.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Error("failed running the server", slog.Any("err", err))
	}
}
