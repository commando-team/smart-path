package router

import (
	"testing"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetBestChannel(t *testing.T) {
	bestChannel := &domain.Channel{
		Id:             uuid.New(),
		Name:           "Best Channel (Low fee, high success)",
		IsActive:       true,
		FeePercent:     1,
		SuccessPercent: 95,
		LatencyMs:      50,
	}

	worstChannel := &domain.Channel{
		Id:             uuid.New(),
		Name:           "Worst Channel (High fee, low success)",
		IsActive:       true,
		FeePercent:     5,
		SuccessPercent: 70,
		LatencyMs:      300,
	}

	inactiveChannel := &domain.Channel{
		Id:             uuid.New(),
		Name:           "Inactive Channel but theoretically perfect",
		IsActive:       false,
		FeePercent:     0,
		SuccessPercent: 100,
		LatencyMs:      10,
	}

	t.Run("should select the active channel with the lowest score", func(t *testing.T) {
		channels := []*domain.Channel{worstChannel, bestChannel}

		got := GetBestChannel(channels)
		assert.Equal(t, got, bestChannel)
	})

	t.Run("should ignore inactive channels", func(t *testing.T) {
		channels := []*domain.Channel{inactiveChannel, worstChannel}

		got := GetBestChannel(channels)
		assert.Equal(t, got, worstChannel)
	})

	t.Run("should return nil if no channels are available", func(t *testing.T) {
		channels := []*domain.Channel{inactiveChannel}

		got := GetBestChannel(channels)
		assert.Nil(t, got)
	})
}
