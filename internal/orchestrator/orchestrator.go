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

type Orchestrator struct {
	log      *slog.Logger
	repo     PaymentRepository
	channels []*domain.Channel
}

func New(log *slog.Logger, repo PaymentRepository, channels []*domain.Channel) *Orchestrator {
	return &Orchestrator{
		log:      log,
		repo:     repo,
		channels: channels,
	}
}

func (o *Orchestrator) ProcessPayment(ctx context.Context, payment *db.Payment) error {
	o.log.Info("starting payment processing", slog.String("payment_id", payment.ID.String()))

	triedChannels := make(map[string]bool)

	for {
		// статус -> routing
		if err := o.repo.UpdateStatus(ctx, payment.ID, StatusRouting); err != nil {
			return fmt.Errorf("failed to update status to ROUTING: %w", err)
		}

		// выбираем лучший канал среди еще не опробованных
		selectedChannel := o.selectBestUntriedChannel(triedChannels)

		if selectedChannel == nil {
			o.log.Warn("no available channels left for payment", slog.String("payment_id", payment.ID.String()))
			return o.repo.UpdateStatus(ctx, payment.ID, StatusFailure)
		}

		triedChannels[selectedChannel.Id.String()] = true

		// статус -> processing
		if err := o.repo.UpdateStatus(ctx, payment.ID, StatusProcessing); err != nil {
			return err
		}

		start := time.Now()
		result, simErr := channel.SimulatePayment(ctx, selectedChannel, payment)
		latency := time.Since(start)

		logWithParams := o.log.With(
			slog.String("payment_id", payment.ID.String()),
			slog.String("channel", selectedChannel.Name),
			slog.Int64("latency ms", latency.Milliseconds()),
		)

		switch result {
		case channel.Success:
			logWithParams.Info("payment successful")
			// TODO: record success
			return o.repo.UpdateStatus(ctx, payment.ID, StatusSuccess)

		case channel.InsufficientFunds:
			logWithParams.Warn("insufficient funds, failing payment", logger.Err(simErr))
			return o.repo.UpdateStatus(ctx, payment.ID, StatusFailure)

		case channel.Timeout:
			// канал завис: переходим в indeterminate и делаем сверку статуса
			logWithParams.Warn("channel timeout, checking status...")
			_ = o.repo.UpdateStatus(ctx, payment.ID, StatusIndeterminate)

			checkResult, _ := channel.SimulateCheckStatus(ctx, selectedChannel, payment)
			if checkResult == channel.Success {
				// сверка показала, что деньги на самом деле списались
				// TODO: record success
				return o.repo.UpdateStatus(ctx, payment.ID, StatusSuccess)
			}

			// TODO: record failure
			// цикл продолжается, попробуем следующий канал

		case channel.TechnicalError:
			// канал упал с 500 ошибкой: штрафуем и идем на retry
			logWithParams.Warn("technical failure, retrying with next channel", logger.Err(simErr))
			// TODO: record failure
			// цикл продолжается, попробуем следующий канал
		}
	}
}

func (o *Orchestrator) selectBestUntriedChannel(tried map[string]bool) *domain.Channel {
	var candidates []*domain.Channel
	for _, ch := range o.channels {
		if !tried[ch.Id.String()] {
			candidates = append(candidates, ch)
		}
	}
	return router.GetBestChannel(candidates)
}
