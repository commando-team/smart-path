package repository

import (
	"context"

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
	paymentStatus.Scan(status)

	return r.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{
		ID:     paymentID,
		Status: paymentStatus,
	})
}
