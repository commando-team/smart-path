package service

import (
	"context"
	"log/slog"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator"
	"github.com/google/uuid"
)

type PaymentRepository interface {
	CreatePayment(ctx context.Context, userID uuid.UUID, amount int64, status string) (uuid.UUID, error)
	GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error)
}

type PaymentService struct {
	log          *slog.Logger
	repo         PaymentRepository
	orchestrator *orchestrator.Orchestrator
}

func NewPaymentService(log *slog.Logger, repo PaymentRepository, orchestrator *orchestrator.Orchestrator) *PaymentService {
	return &PaymentService{
		log:          log,
		repo:         repo,
		orchestrator: orchestrator,
	}
}

func (s *PaymentService) Pay(ctx context.Context, userID uuid.UUID, amount int64) error {
	paymentID, err := s.repo.CreatePayment(ctx, userID, amount, orchestrator.StatusPending)
	if err != nil {
		s.log.Error("failed to create payment", logger.Err(err))
	}

	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		s.log.Error("failed to find payment", logger.Err(err))
	}

	err = s.orchestrator.ProcessPayment(ctx, &payment)

	if err != nil {
		s.log.Error("orchestrator error", logger.Err(err))
	}

	return nil
}

func (s *PaymentService) GetPayment(ctx context.Context, paymentID uuid.UUID) (*db.Payment, error) {
	payment, err := s.repo.GetPayment(ctx, paymentID)
	if err != nil {
		s.log.Error("failed to find payment", logger.Err(err))

		return &db.Payment{}, err
	}

	return &payment, nil
}
