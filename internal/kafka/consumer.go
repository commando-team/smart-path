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
	"github.com/segmentio/kafka-go"
)

type PaymentRepository interface {
	GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error)
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
		Brokers:        brokers,
		GroupID:        groupID,
		Topic:          topic,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: 0,
	})

	return &Consumer{
		reader:       reader,
		log:          log,
		repo:         repo,
		orchestrator: orch,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			c.log.Error("failed to fetch kafka message", logger.Err(err))
			time.Sleep(200 * time.Millisecond)
			continue
		}

		if err := c.process(ctx, msg); err != nil {
			c.log.Error("failed to process message", logger.Err(err))
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			c.log.Error("failed to commit message", logger.Err(err))
		}
	}
}

func (c *Consumer) process(ctx context.Context, msg kafka.Message) error {
	var task PaymentMessage
	if err := json.Unmarshal(msg.Value, &task); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	payment, err := c.repo.GetPayment(ctx, task.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to find payment: %w", err)
	}

	if string(payment.Status) != orchestrator.StatusPending {
		return nil
	}

	if err := c.orchestrator.ProcessPayment(ctx, &payment); err != nil {
		return fmt.Errorf("orchestrator processing error: %w", err)
	}

	return nil
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}
