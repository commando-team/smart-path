package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/commando-team/smart-path/internal/config"
	"github.com/commando-team/smart-path/internal/http/handler"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator"
	"github.com/commando-team/smart-path/internal/repository"
	"github.com/commando-team/smart-path/internal/router"
	"github.com/commando-team/smart-path/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/commando-team/smart-path/docs"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

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
	if err := runMigrations(dsn, log); err != nil {
		log.Error("fatal error during migrations", logger.Err(err))
		os.Exit(1)
	}

	// пул соединений к бд
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("db connection error", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()

	paymentRepo := repository.NewPaymentRepository(pool)

	orchestrator := orchestrator.New(log, paymentRepo, router.DefaultChannels)

	paymentService := service.NewPaymentService(log, paymentRepo, orchestrator)

	apiHandler := handler.NewApiHandler(log, paymentService)

	r := gin.Default()

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.POST("/payment", apiHandler.Pay)
	r.GET("/payment/:id", apiHandler.GetPayment)

	if err := r.Run(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Error("failed running the server", logger.Err(err))
	}
}

func runMigrations(dsn string, log *slog.Logger) error {
	m, err := migrate.New("file://db/migrations", dsn)
	if err != nil {
		return fmt.Errorf("failed to initialize migration: %w", err)
	}

	defer func() {
		if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
			log.Error("failed to close migration resources", "src_err", srcErr, "db_err", dbErr)
		}
	}()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration error: %w", err)
	}

	return nil
}
