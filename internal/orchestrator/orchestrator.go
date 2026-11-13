package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/commando-team/smart-path/internal/channel"
	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/domain"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/router"
	"github.com/google/uuid"
)

const (
	StatusPending       = "pending"
	StatusRouting       = "routing"
	StatusProcessing    = "processing"
	StatusSuccess       = "success"
	StatusFailure       = "failure"
	StatusIndeterminate = "indeterminate"
)

type PaymentRepository interface {
	UpdateStatus(ctx context.Context, paymentID uuid.UUID, status string) error
}

type ChannelRepository interface {
	GetActiveChannels(ctx context.Context) ([]*domain.Channel, error)
	RecordAttempt(ctx context.Context, channelID uuid.UUID, success bool, latencyMs int64) error
}

type Orchestrator struct {
	log         *slog.Logger
	paymentRepo PaymentRepository
	channelRepo ChannelRepository
}

func New(log *slog.Logger, paymentRepo PaymentRepository, channelRepo ChannelRepository) *Orchestrator {
	return &Orchestrator{
		log:         log,
		paymentRepo: paymentRepo,
		channelRepo: channelRepo,
	}
}

func (o *Orchestrator) ProcessPayment(ctx context.Context, payment *db.Payment) error {
	o.log.Info("starting payment processing", slog.String("payment_id", payment.ID.String()))

	channels, err := o.channelRepo.GetActiveChannels(ctx)
	if err != nil {
		o.log.Error("failed to load channels", logger.Err(err))
		return o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusFailure)
	}

	triedChannels := make(map[string]bool)

	for {
		if err := o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusRouting); err != nil {
			return fmt.Errorf("failed to update status to ROUTING: %w", err)
		}

		selectedChannel := o.selectBestUntriedChannel(channels, triedChannels)

		if selectedChannel == nil {
			o.log.Warn("no available channels left for payment", slog.String("payment_id", payment.ID.String()))
			return o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusFailure)
		}

		triedChannels[selectedChannel.Id.String()] = true

		if err := o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusProcessing); err != nil {
			return err
		}

		start := time.Now()
		result, simErr := channel.SimulatePayment(ctx, selectedChannel, payment)
		latency := time.Since(start)

		logWithParams := o.log.With(
			slog.String("payment_id", payment.ID.String()),
			slog.String("channel", selectedChannel.Name),
			slog.Int64("latency_ms", latency.Milliseconds()),
		)

		switch result {
		case channel.Success:
			logWithParams.Info("payment successful")
			_ = o.channelRepo.RecordAttempt(ctx, selectedChannel.Id, true, latency.Milliseconds())
			return o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusSuccess)

		case channel.InsufficientFunds:
			logWithParams.Warn("insufficient funds, failing payment", logger.Err(simErr))
			_ = o.channelRepo.RecordAttempt(ctx, selectedChannel.Id, true, latency.Milliseconds())
			return o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusFailure)

		case channel.Timeout:
			logWithParams.Warn("channel timeout, checking status...")
			_ = o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusIndeterminate)

			checkResult, _ := channel.SimulateCheckStatus(ctx, selectedChannel, payment)
			if checkResult == channel.Success {
				_ = o.channelRepo.RecordAttempt(ctx, selectedChannel.Id, true, latency.Milliseconds())
				return o.paymentRepo.UpdateStatus(ctx, payment.ID, StatusSuccess)
			}

			_ = o.channelRepo.RecordAttempt(ctx, selectedChannel.Id, false, latency.Milliseconds())

		case channel.TechnicalError:
			logWithParams.Warn("technical failure, retrying with next channel", logger.Err(simErr))
			_ = o.channelRepo.RecordAttempt(ctx, selectedChannel.Id, false, latency.Milliseconds())
		}
	}
}

func (o *Orchestrator) selectBestUntriedChannel(channels []*domain.Channel, tried map[string]bool) *domain.Channel {
	var candidates []*domain.Channel
	for _, ch := range channels {
		if !tried[ch.Id.String()] {
			candidates = append(candidates, ch)
		}
	}
	return router.GetBestChannel(candidates)
}
