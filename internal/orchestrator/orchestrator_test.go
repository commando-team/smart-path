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
)

func TestOrchestrator_ProcessPayment_SuccessOnFirstChannel(t *testing.T) {
	discardLogger := logger.NewDiscardLogger()

	successChannel := &domain.Channel{
		Id:             uuid.New(),
		Name:           "100% Success Channel",
		IsActive:       true,
		FeePercent:     0,
		SuccessPercent: 100,
		LatencyMs:      0,
	}

	channels := []*domain.Channel{successChannel}
	payment := &db.Payment{
		ID:     uuid.New(),
		Amount: 1000,
	}

	mockRepo := mocks.NewMockPaymentRepository(t)

	ctx := context.Background()
	mockRepo.On("UpdateStatus", ctx, payment.ID, StatusRouting).Return(nil).Once()
	mockRepo.On("UpdateStatus", ctx, payment.ID, StatusProcessing).Return(nil).Once()
	mockRepo.On("UpdateStatus", ctx, payment.ID, StatusSuccess).Return(nil).Once()

	orch := New(discardLogger, mockRepo, channels)
	err := orch.ProcessPayment(ctx, payment)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

func TestOrchestrator_ProcessPayment_NoChannelsLeft(t *testing.T) {
	discardLogger := logger.NewDiscardLogger()

	var channels []*domain.Channel
	payment := &db.Payment{
		ID: uuid.New(),
	}

	mockRepo := mocks.NewMockPaymentRepository(t)
	ctx := context.Background()

	mockRepo.On("UpdateStatus", ctx, payment.ID, StatusRouting).Return(nil).Once()
	mockRepo.On("UpdateStatus", ctx, payment.ID, StatusFailure).Return(nil).Once()

	orch := New(discardLogger, mockRepo, channels)
	err := orch.ProcessPayment(ctx, payment)

	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
