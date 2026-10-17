package router

import (
	"cmp"
	"slices"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/google/uuid"
)

const (
	WeightFee     = 100
	WeightLatency = 1
	WeightFailure = 200
)

var DefaultChannels = []*domain.Channel{
	{
		Id:             uuid.New(),
		Name:           "T-Bank",
		IsActive:       true,
		FeePercent:     0,
		SuccessPercent: 99,
		LatencyMs:      100,
	},
	{
		Id:             uuid.New(),
		Name:           "Sberbank",
		IsActive:       true,
		FeePercent:     10,
		SuccessPercent: 80,
		LatencyMs:      500,
	},
}

func GetBestChannel(channels []*domain.Channel) *domain.Channel {
	var activeChannels []*domain.Channel

	for _, channel := range channels {
		if channel.IsActive {
			activeChannels = append(activeChannels, channel)
		}
	}

	if len(activeChannels) == 0 {
		return nil
	}

	slices.SortFunc(activeChannels, func(a, b *domain.Channel) int {
		return cmp.Compare(calculateScore(a), calculateScore(b))
	})

	return activeChannels[0]
}

func calculateScore(channel *domain.Channel) int64 {
	failurePercent := 100 - channel.SuccessPercent

	return (channel.FeePercent * WeightFee) +
		(channel.LatencyMs * WeightLatency) +
		(failurePercent * WeightFailure)
}
