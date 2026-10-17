package handler

import (
	"context"
	"log/slog"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/google/uuid"
)

type PaymentService interface {
	Pay(ctx context.Context, userID uuid.UUID, amount int64) error
	GetPayment(ctx context.Context, paymentID uuid.UUID) (*db.Payment, error)
}

type ApiHandler struct {
	log     *slog.Logger
	payment PaymentService
}

func NewApiHandler(log *slog.Logger, payment PaymentService) *ApiHandler {
	return &ApiHandler{
		log:     log,
		payment: payment,
	}
}
