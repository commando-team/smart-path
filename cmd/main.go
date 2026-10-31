package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/commando-team/smart-path/internal/config"
	"github.com/commando-team/smart-path/internal/http/handler"
	"github.com/commando-team/smart-path/internal/kafka"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser, cfg.DBPassword, cfg.DBHost, cfg.DBPort, cfg.DBName,
	)

	if err := runMigrations(dsn, log); err != nil {
		log.Error("fatal error during migrations", logger.Err(err))
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("db connection error", logger.Err(err))
		os.Exit(1)
	}
	defer pool.Close()

	kafkaBrokers := []string{cfg.KafkaBroker}
	kafkaTopic := "payment.process"
	consumerGroup := "payment-orchestrator-group"

	producer := kafka.NewProducer(kafkaBrokers, kafkaTopic)
	defer func() {
		if err := producer.Close(); err != nil {
			log.Error("failed to close kafka producer", logger.Err(err))
		}
	}()

	paymentRepo := repository.NewPaymentRepository(pool)
	orch := orchestrator.New(log, paymentRepo, router.DefaultChannels)

	consumer := kafka.NewConsumer(kafkaBrokers, kafkaTopic, consumerGroup, log, paymentRepo, orch)
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("failed to close kafka consumer", logger.Err(err))
		}
	}()

	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Error("consumer stopped with error", logger.Err(err))
		}
	}()

	paymentService := service.NewPaymentService(log, paymentRepo, producer)
	apiHandler := handler.NewApiHandler(log, paymentService)

	r := gin.Default()
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.POST("/payment", apiHandler.Pay)
	r.GET("/payment/:id", apiHandler.GetPayment)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("failed running the server", logger.Err(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", logger.Err(err))
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
