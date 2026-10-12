package service

import (
	"context"
	"log/slog"
)

type PaymentService struct{}

func NewPaymentService(log *slog.Logger) *PaymentService {
	return &PaymentService{}
}

func (s *PaymentService) Pay(ctx context.Context) error {
	return nil
}
