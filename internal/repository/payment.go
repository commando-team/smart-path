package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (r *PaymentRepository) CreatePayment(
	ctx context.Context,
	userID uuid.UUID,
	amount int64,
	status string,
	idempotencyKey string,
) (db.Payment, bool, error) {
	var paymentStatus db.PaymentStatus
	if err := paymentStatus.Scan(status); err != nil {
		return db.Payment{}, false, fmt.Errorf("failed to scan status: %w", err)
	}

	key := pgtype.Text{
		String: idempotencyKey,
		Valid:  idempotencyKey != "",
	}

	payment, err := r.queries.CreatePayment(ctx, db.CreatePaymentParams{
		UserID:         userID,
		Amount:         amount,
		Status:         paymentStatus,
		IdempotencyKey: key,
	})
	if err == nil {
		return payment, true, nil
	}

	if !errors.Is(err, pgx.ErrNoRows) || !key.Valid {
		return db.Payment{}, false, fmt.Errorf("create payment: %w", err)
	}

	existing, getErr := r.queries.GetPaymentByIdempotencyKey(ctx, key)
	if getErr != nil {
		return db.Payment{}, false, fmt.Errorf("get payment by idempotency key: %w", getErr)
	}

	return existing, false, nil
}

func (r *PaymentRepository) GetPayment(ctx context.Context, paymentID uuid.UUID) (db.Payment, error) {
	return r.queries.GetPayment(ctx, paymentID)
}

func (r *PaymentRepository) ClaimPaymentForRouting(ctx context.Context, paymentID uuid.UUID) (db.Payment, error) {
	return r.queries.ClaimPaymentForRouting(ctx, paymentID)
}
