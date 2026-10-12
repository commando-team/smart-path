package channel

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/domain"
)

type ChannelResult int64

const (
	Success ChannelResult = iota
	Failure
	Timeout
	TechnicalError
	InsufficientFunds
)

var (
	ErrGatewayTimeout    = errors.New("gateway timeout")
	ErrInternalGateway   = errors.New("gateway internal error (500)")
	ErrInsufficientFunds = errors.New("insufficient client funds")
)

func SimulatePayment(ctx context.Context, channel *domain.Channel, payment *db.Payment) (ChannelResult, error) {
	if channel.LatencyMs > 0 {
		select {
		case <-time.After(time.Duration(channel.LatencyMs) * time.Millisecond):
		case <-ctx.Done():
			return Timeout, ctx.Err()
		}
	}

	roll := rand.Intn(100) + 1

	if int64(roll) <= channel.SuccessPercent {
		return Success, nil
	}

	failureReasonRoll := rand.Intn(100)

	switch {
	case failureReasonRoll < 50:
		return TechnicalError, ErrInternalGateway

	case failureReasonRoll < 80:
		return Timeout, ErrGatewayTimeout

	default:
		return InsufficientFunds, ErrInsufficientFunds
	}
}

func SimulateCheckStatus(ctx context.Context, channel *domain.Channel, payment *db.Payment) (ChannelResult, error) {
	time.Sleep(50 * time.Millisecond)

	if rand.Intn(2) == 0 {
		return Success, nil
	}
	return Failure, nil
}
