package handler

import (
	"context"
	"log/slog"
)

type PaymentService interface {
	Pay(ctx context.Context) error
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
