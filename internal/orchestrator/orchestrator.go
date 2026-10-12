package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/commando-team/smart-path/internal/channel"
	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/domain"
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
		// 1. Статус -> ROUTING
		if err := o.repo.UpdateStatus(ctx, payment.ID, StatusRouting); err != nil {
			return fmt.Errorf("failed to update status to ROUTING: %w", err)
		}

		// 2. Выбираем лучший канал среди еще не опробованных
		selectedChannel := o.selectBestUntriedChannel(triedChannels)

		// Если доступных каналов больше нет — это терминальный FAILURE
		if selectedChannel == nil {
			o.log.Warn("no available channels left for payment", "payment_id", payment.ID)
			return o.repo.UpdateStatus(ctx, payment.ID, StatusFailure)
		}

		triedChannels[selectedChannel.Id.String()] = true

		// 3. Статус -> PROCESSING + запись о попытке
		if err := o.repo.UpdateStatus(ctx, payment.ID, StatusProcessing); err != nil {
			return err
		}

		// routingReason := fmt.Sprintf("chosen by score: %d (fee: %d%%)",
		// 	selectedChannel.CalculateScore(), selectedChannel.FeePercent)

		// attemptID, err := o.repo.CreateAttempt(ctx, payment.ID, selectedChannel.Id, routingReason)
		// if err != nil {
		// 	return fmt.Errorf("failed to create attempt: %w", err)
		// }

		// 4. Отправляем запрос в канал (с замером времени)
		// start := time.Now()
		result, _ /*simErr*/ := channel.SimulatePayment(ctx, selectedChannel, payment)
		// latency := time.Since(start)

		// 5. Разбор результата в соответствии со схемой FSM
		switch result {
		case channel.Success:
			o.log.Info("payment successful", "payment_id", payment.ID, "channel", selectedChannel.Name)
			// selectedChannel.RecordSuccess(latency)
			// _ = o.repo.CompleteAttempt(ctx, attemptID, "SUCCESS", nil)
			return o.repo.UpdateStatus(ctx, payment.ID, StatusSuccess)

		case channel.InsufficientFunds:
			// Бизнес-отказ: у клиента нет денег. Пробовать другие каналы бессмысленно!
			o.log.Warn("insufficient funds, failing payment", "payment_id", payment.ID)
			// errMsg := simErr.Error()
			// _ = o.repo.CompleteAttempt(ctx, attemptID, "FAILED", &errMsg)
			return o.repo.UpdateStatus(ctx, payment.ID, StatusFailure)

		case channel.Timeout:
			// Канал завис: переходим в INDETERMINATE и делаем сверку статуса
			o.log.Warn("channel timeout, checking status...", "payment_id", payment.ID, "channel", selectedChannel.Name)
			_ = o.repo.UpdateStatus(ctx, payment.ID, StatusIndeterminate)

			checkResult, _ := channel.SimulateCheckStatus(ctx, selectedChannel, payment)
			if checkResult == channel.Success {
				// Сверка показала, что деньги на самом деле списались
				// selectedChannel.RecordSuccess(latency)
				// _ = o.repo.CompleteAttempt(ctx, attemptID, "SUCCESS", nil)
				return o.repo.UpdateStatus(ctx, payment.ID, StatusSuccess)
			}

			// Если даже после сверки не подтвердилось — фиксируем сбой канала и идем на retry
			// selectedChannel.RecordFailure()
			// errMsg := "timeout confirmed after check"
			// _ = o.repo.CompleteAttempt(ctx, attemptID, "TIMEOUT", &errMsg)
			// Цикл for продолжается: попробуем следующий канал

		case channel.TechnicalError:
			// Шлюз упал с 500 ошибкой: штрафуем канал и идем на retry
			o.log.Warn("technical failure, retrying with next channel", "payment_id", payment.ID, "channel", selectedChannel.Name)
			// selectedChannel.RecordFailure()
			// errMsg := simErr.Error()
			// _ = o.repo.CompleteAttempt(ctx, attemptID, "FAILED", &errMsg)
			// Цикл for продолжается: попробуем следующий канал
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
