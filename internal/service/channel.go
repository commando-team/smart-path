package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/google/uuid"
)

type ChannelRepository interface {
	ListChannels(ctx context.Context) ([]*domain.Channel, error)
	GetActiveChannels(ctx context.Context) ([]*domain.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*domain.Channel, error)
	CreateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error)
	UpdateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error)
	RecordAttempt(ctx context.Context, channelID uuid.UUID, success bool, latencyMs int64) error
}

type ChannelService struct {
	log  *slog.Logger
	repo ChannelRepository
}

func NewChannelService(log *slog.Logger, repo ChannelRepository) *ChannelService {
	return &ChannelService{
		log:  log,
		repo: repo,
	}
}

func (s *ChannelService) ListChannels(ctx context.Context) ([]*domain.Channel, error) {
	channels, err := s.repo.ListChannels(ctx)
	if err != nil {
		s.log.Error("failed to list channels", logger.Err(err))
		return nil, fmt.Errorf("list channels: %w", err)
	}
	return channels, nil
}

func (s *ChannelService) GetChannel(ctx context.Context, id uuid.UUID) (*domain.Channel, error) {
	ch, err := s.repo.GetChannel(ctx, id)
	if err != nil {
		s.log.Error("failed to get channel", slog.String("id", id.String()), logger.Err(err))
		return nil, fmt.Errorf("get channel: %w", err)
	}
	return ch, nil
}

func (s *ChannelService) CreateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error) {
	created, err := s.repo.CreateChannel(ctx, ch)
	if err != nil {
		s.log.Error("failed to create channel", logger.Err(err))
		return nil, fmt.Errorf("create channel: %w", err)
	}
	s.log.Info("channel created", slog.String("id", created.Id.String()), slog.String("name", created.Name))
	return created, nil
}

func (s *ChannelService) UpdateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error) {
	updated, err := s.repo.UpdateChannel(ctx, ch)
	if err != nil {
		s.log.Error("failed to update channel", slog.String("id", ch.Id.String()), logger.Err(err))
		return nil, fmt.Errorf("update channel: %w", err)
	}
	s.log.Info("channel updated", slog.String("id", updated.Id.String()), slog.String("name", updated.Name))
	return updated, nil
}
