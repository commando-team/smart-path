package repository

import (
	"context"
	"fmt"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PaymentRepository struct {
	queries *db.Queries
}

func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{queries: db.New(pool)}
}

func (r *PaymentRepository) UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string) error {
	var paymentStatus db.PaymentStatus
	if err := paymentStatus.Scan(status); err != nil {
		return fmt.Errorf("failed to scan status: %w", err)
	}

	return r.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:     paymentID,
		Status: paymentStatus,
	})
}

func (r *PaymentRepository) CreatePayment(ctx context.Context, userID uuid.UUID, amount int64, status string) (uuid.UUID, error) {
	var paymentStatus db.PaymentStatus
	if err := paymentStatus.Scan(status); err != nil {
		return uuid.Nil, fmt.Errorf("failed to scan status: %w", err)
	}

	return r.queries.CreatePayment(ctx, db.CreatePaymentParams{
		UserID: userID,
		Amount: amount,
		Status: paymentStatus,
	})
}

func (r *PaymentRepository) GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error) {
	return r.queries.GetPayment(ctx, paymentID)
}
