package kafka

import (
	"time"

	"github.com/google/uuid"
)

type PaymentMessage struct {
	PaymentID uuid.UUID `json:"payment_id"`
	UserID    uuid.UUID `json:"user_id"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}
