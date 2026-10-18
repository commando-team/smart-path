package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/kafka"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator"
	"github.com/google/uuid"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, amount int64, status string) (uuid.UUID, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error)
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

func (s *PaymentService) Pay(ctx context.Context, userID uuid.UUID, amount int64) (uuid.UUID, error) {
	paymentID, err := s.repo.CreatePayment(ctx, userID, amount, orchestrator.StatusPending)
	if err != nil {
		s.log.Error("failed to create payment", logger.Err(err))
		return uuid.Nil, fmt.Errorf("create payment: %w", err)
	}

	msg := kafka.PaymentMessage{
		PaymentID: paymentID,
		UserID:    userID,
		Amount:    amount,
		CreatedAt: time.Now(),
	}

	if err := s.producer.SendPayment(ctx, msg); err != nil {
		s.log.Error("failed to send payment to kafka", logger.Err(err))
		return uuid.Nil, fmt.Errorf("produce payment: %w", err)
	}

	return paymentID, nil
}

func (s *PaymentService) GetPayment(ctx context.Context, paymentID uuid.UUID) (*db.Payment, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		s.log.Error("failed to find payment", logger.Err(err))
		return &db.Payment{}, err
	}

	return &payment, nil
}
