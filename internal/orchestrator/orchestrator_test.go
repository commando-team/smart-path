package orchestrator

import (
	"context"
	"testing"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/domain"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/orchestrator/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestOrchestrator_ProcessPayment_SuccessOnFirstChannel(t *testing.T) {
	discardLogger := logger.NewDiscardLogger()

	channelID := uuid.New()
	successChannel := &domain.Channel{
		Id:             channelID,
		Name:           "100% Success Channel",
		IsActive:       true,
		FeePercent:     0,
		SuccessPercent: 100,
		LatencyMs:      0,
	}

	payment := &db.Payment{
		ID:     uuid.New(),
		Amount: 1000,
	}

	mockPaymentRepo := mocks.NewMockPaymentRepository(t)
	mockChannelRepo := mocks.NewMockChannelRepository(t)

	ctx := context.Background()
	mockChannelRepo.On("GetActiveChannels", ctx).Return([]*domain.Channel{successChannel}, nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, payment.ID, StatusRouting).Return(nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, payment.ID, StatusProcessing).Return(nil).Once()
	mockChannelRepo.On("RecordAttempt", ctx, channelID, true, mock.AnythingOfType("int64")).Return(nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, payment.ID, StatusSuccess).Return(nil).Once()

	orch := New(discardLogger, mockPaymentRepo, mockChannelRepo)
	err := orch.ProcessPayment(ctx, payment)

	assert.NoError(t, err)

	mockPaymentRepo.AssertExpectations(t)
	mockChannelRepo.AssertExpectations(t)
}

func TestOrchestrator_ProcessPayment_NoChannelsLeft(t *testing.T) {
	discardLogger := logger.NewDiscardLogger()

	payment := &db.Payment{
		ID: uuid.New(),
	}

	mockPaymentRepo := mocks.NewMockPaymentRepository(t)
	mockChannelRepo := mocks.NewMockChannelRepository(t)
	ctx := context.Background()

	mockChannelRepo.On("GetActiveChannels", ctx).Return([]*domain.Channel{}, nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, payment.ID, StatusRouting).Return(nil).Once()
	mockPaymentRepo.On("UpdateStatus", ctx, payment.ID, StatusFailure).Return(nil).Once()

	orch := New(discardLogger, mockPaymentRepo, mockChannelRepo)
	err := orch.ProcessPayment(ctx, payment)

	assert.NoError(t, err)

	mockPaymentRepo.AssertExpectations(t)
	mockChannelRepo.AssertExpectations(t)
}
