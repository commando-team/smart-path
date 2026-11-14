package repository

import (
	"context"
	"fmt"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelRepository struct {
	queries *db.Queries
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{queries: db.New(pool)}
}

func (r *ChannelRepository) ListChannels(ctx context.Context) ([]*domain.Channel, error) {
	rows, err := r.queries.ListChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}

	result := make([]*domain.Channel, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainChannel(row))
	}
	return result, nil
}

func (r *ChannelRepository) GetActiveChannels(ctx context.Context) ([]*domain.Channel, error) {
	rows, err := r.queries.ListActiveChannels(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active channels: %w", err)
	}

	result := make([]*domain.Channel, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainChannel(row))
	}
	return result, nil
}

func (r *ChannelRepository) GetChannel(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	row, err := r.queries.GetChannelByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get channel: %w", err)
	}
	return toDomainChannel(row), nil
}

func (r *ChannelRepository) CreateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error) {
	row, err := r.queries.CreateChannel(ctx, db.CreateChannelParams{
		Name:           ch.Name,
		IsActive:       ch.IsActive,
		FeePercent:     ch.FeePercent,
		SuccessPercent: ch.SuccessPercent,
		LatencyMs:      ch.LatencyMs,
	})
	if err != nil {
		return nil, fmt.Errorf("create channel: %w", err)
	}
	return toDomainChannel(row), nil
}

func (r *ChannelRepository) UpdateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error) {
	row, err := r.queries.UpdateChannel(ctx, db.UpdateChannelParams{
		ID:             ch.Id,
		Name:           ch.Name,
		IsActive:       ch.IsActive,
		FeePercent:     ch.FeePercent,
		SuccessPercent: ch.SuccessPercent,
		LatencyMs:      ch.LatencyMs,
	})
	if err != nil {
		return nil, fmt.Errorf("update channel: %w", err)
	}
	return toDomainChannel(row), nil
}

func (r *ChannelRepository) RecordAttempt(ctx context.Context, channelID uuid.UUID, success bool, latencyMs int64) error {
	ch, err := r.queries.GetChannelByID(ctx, channelID)
	if err != nil {
		return fmt.Errorf("get channel for metrics update: %w", err)
	}

	totalReq := ch.TotalRequests + 1
	failedReq := ch.FailedRequests
	consecutiveFail := ch.ConsecutiveFailures
	isActive := ch.IsActive

	newLatency := ch.LatencyMs
	if newLatency <= 0 {
		newLatency = latencyMs
	} else {
		newLatency = (70*ch.LatencyMs + 30*latencyMs) / 100
	}

	var newSuccessPercent int64
	if success {
		consecutiveFail = 0
		newSuccessPercent = (80*ch.SuccessPercent + 20*100) / 100
	} else {
		failedReq++
		consecutiveFail++
		newSuccessPercent = (80*ch.SuccessPercent + 20*0) / 100
		if consecutiveFail >= 5 {
			isActive = false
		}
	}

	if newSuccessPercent < 0 {
		newSuccessPercent = 0
	}
	if newSuccessPercent > 100 {
		newSuccessPercent = 100
	}

	return r.queries.UpdateChannelMetrics(ctx, db.UpdateChannelMetricsParams{
		ID:                  channelID,
		TotalRequests:       totalReq,
		FailedRequests:      failedReq,
		ConsecutiveFailures: consecutiveFail,
		SuccessPercent:      newSuccessPercent,
		LatencyMs:           newLatency,
		IsActive:            isActive,
	})
}

func toDomainChannel(c db.Channel) *domain.Channel {
	return &domain.Channel{
		Id:                  c.ID,
		Name:                c.Name,
		IsActive:            c.IsActive,
		FeePercent:          c.FeePercent,
		SuccessPercent:      c.SuccessPercent,
		LatencyMs:           c.LatencyMs,
		TotalRequests:       c.TotalRequests,
		FailedRequests:      c.FailedRequests,
		ConsecutiveFailures: c.ConsecutiveFailures,
	}
}
