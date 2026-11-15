package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/kafka"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator"
	"github.com/google/uuid"
)

var ErrIdempotencyConflict = errors.New("idempotency key already used with different payment parameters")

type PayResult struct {
	PaymentID uuid.UUID
	Status    db.PaymentStatus
	Created   bool
}

type PaymentRepository interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, amount int64, status string, idempotencyKey string) (db.Payment, bool, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error)
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string) error
}

type PaymentProducer interface {
	SendPayment(ctx context.Context, msg kafka.PaymentMessage) error
}

type PaymentService struct {
	log      *slog.Logger
	repo     PaymentRepository
	producer PaymentProducer
}

func NewPaymentService(log *slog.Logger, repo PaymentRepository, producer PaymentProducer) *PaymentService {
	return &PaymentService{
		log:      log,
		repo:     repo,
		producer: producer,
	}
}

func (s *PaymentService) Pay(ctx context.Context, userID uuid.UUID, amount int64, idempotencyKey string) (*PayResult, error) {
	payment, created, err := s.repo.CreatePayment(ctx, userID, amount, orchestrator.StatusPending, idempotencyKey)
	if err != nil {
		s.log.Error("failed to create payment", logger.Err(err))
		return nil, fmt.Errorf("create payment: %w", err)
	}

	if !created {
		if payment.UserID != userID || payment.Amount != amount {
			s.log.Warn("idempotency conflict",
				slog.String("idempotency_key", idempotencyKey),
				slog.String("existing_payment_id", payment.ID.String()),
			)
			return nil, ErrIdempotencyConflict
		}

		s.log.Info("duplicate payment request deduplicated",
			slog.String("idempotency_key", idempotencyKey),
			slog.String("payment_id", payment.ID.String()),
			slog.String("status", string(payment.Status)),
		)
		return &PayResult{
			PaymentID: payment.ID,
			Status:    payment.Status,
			Created:   false,
		}, nil
	}

	msg := kafka.PaymentMessage{
		PaymentID: payment.ID,
		UserID:    userID,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	if err := s.producer.SendPayment(ctx, msg); err != nil {
		s.log.Error("failed to send payment to kafka", logger.Err(err))

		failCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		if updateErr := s.repo.UpdateStatus(failCtx, payment.ID, orchestrator.StatusFailure); updateErr != nil {
			s.log.Error("failed to mark payment as failure after kafka error",
				slog.String("payment_id", payment.ID.String()),
				logger.Err(updateErr),
			)
		}

		return nil, fmt.Errorf("produce payment: %w", err)
	}

	return &PayResult{
		PaymentID: payment.ID,
		Status:    payment.Status,
		Created:   true,
	}, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, paymentID uuid.UUID) (*db.Payment, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		s.log.Error("failed to find payment", logger.Err(err))
		return &db.Payment{}, err
	}

	return &payment, nil
}
