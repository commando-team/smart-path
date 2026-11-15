package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/segmentio/kafka-go"
)

var errPermanent = errors.New("permanent message error")

type PaymentRepository interface {
	ClaimPaymentForRouting(ctx context.Context, paymentID uuid.UUID) (db.Payment, error)
}

type Consumer struct {
	reader       *kafka.Reader
	log          *slog.Logger
	repo         PaymentRepository
	orchestrator *orchestrator.Orchestrator
}

func NewConsumer(
	brokers []string,
	topic string,
	groupID string,
	log *slog.Logger,
	repo PaymentRepository,
	orch *orchestrator.Orchestrator,
) *Consumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:               brokers,
		GroupID:               groupID,
		Topic:                 topic,
		MinBytes:              1,
		MaxBytes:              10e6,
		MaxWait:               500 * time.Millisecond,
		WatchPartitionChanges: true,
	})

	return &Consumer{
		reader:       reader,
		log:          log,
		repo:         repo,
		orchestrator: orch,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	c.log.Info("kafka consumer started!")

	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("failed to fetch kafka message", logger.Err(err))
			select {
			case <-time.After(200 * time.Millisecond):
			case <-ctx.Done():
				return nil
			}
			continue
		}

		if err := c.process(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("failed to process message",
				slog.Int("partition", msg.Partition),
				slog.Int64("offset", msg.Offset),
				logger.Err(err),
			)
			if !errors.Is(err, errPermanent) {
				select {
				case <-time.After(200 * time.Millisecond):
				case <-ctx.Done():
					return nil
				}
				continue
			}
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("failed to commit message", logger.Err(err))
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg kafka.Message) error {
	var task PaymentMessage
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		return fmt.Errorf("%w: failed to unmarshal payload: %v", errPermanent, err)
	}

	payment, err := c.repo.ClaimPaymentForRouting(ctx, task.PaymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.log.Info("payment already claimed or processed, skipping",
				slog.String("payment_id", task.PaymentID.String()),
			)
			return nil
		}
		return fmt.Errorf("failed to claim payment: %w", err)
	}

	if err := c.orchestrator.ProcessPayment(ctx, &payment); err != nil {
		return fmt.Errorf("orchestrator processing error: %w", err)
	}

	return nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
