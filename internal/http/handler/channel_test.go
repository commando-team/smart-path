package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakeChannelService struct {
	channels     []*domain.Channel
	channel      *domain.Channel
	err          error
	createCalled bool
	updateCalled bool
	getCalled    bool
	createdInput *domain.Channel
	updatedInput *domain.Channel
	requestedID  uuid.UUID
}

func (f *fakeChannelService) ListChannels(context.Context) ([]*domain.Channel, error) {
	return f.channels, f.err
}

func (f *fakeChannelService) GetChannel(_ context.Context, id uuid.UUID) (*domain.Channel, error) {
	f.getCalled = true
	f.requestedID = id
	return f.channel, f.err
}

func (f *fakeChannelService) CreateChannel(_ context.Context, ch *domain.Channel) (*domain.Channel, error) {
	f.createCalled = true
	f.createdInput = ch
	if f.channel != nil {
		return f.channel, f.err
	}
	return ch, f.err
}

func (f *fakeChannelService) UpdateChannel(_ context.Context, ch *domain.Channel) (*domain.Channel, error) {
	f.updateCalled = true
	f.updatedInput = ch
	if f.channel != nil {
		return f.channel, f.err
	}
	return ch, f.err
}

func newChannelTestRouter(service *fakeChannelService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewChannelHandler(log, service)
	router := gin.New()
	router.GET("/channels", h.ListChannels)
	router.GET("/channels/:id", h.GetChannel)
	router.POST("/channels", h.CreateChannel)
	router.PUT("/channels/:id", h.UpdateChannel)
	return router
}

func TestChannelAPI_Create(t *testing.T) {
	id := uuid.New()
	service := &fakeChannelService{channel: &domain.Channel{
		Id:             id,
		Name:           "Test Channel",
		IsActive:       false,
		FeePercent:     2,
		SuccessPercent: 100,
		LatencyMs:      50,
	}}
	router := newChannelTestRouter(service)

	req := httptest.NewRequest(
		http.MethodPost,
		"/channels",
		strings.NewReader(`{"name":"Test Channel","isActive":false,"feePercent":2,"successPercent":100,"latencyMs":50}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), id.String())
	assert.True(t, service.createCalled)
	assert.Equal(t, "Test Channel", service.createdInput.Name)
	assert.False(t, service.createdInput.IsActive, "false isActive must be accepted")
}

func TestChannelAPI_CreateInvalidRequests(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"missing name", `{"isActive":true}`},
		{"missing isActive", `{"name":"Test"}`},
		{"negative fee", `{"name":"Test","isActive":true,"feePercent":-1}`},
		{"negative success percent", `{"name":"Test","isActive":true,"successPercent":-1}`},
		{"success percent above 100", `{"name":"Test","isActive":true,"successPercent":101}`},
		{"negative latency", `{"name":"Test","isActive":true,"latencyMs":-1}`},
		{"invalid JSON", `{"name":`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeChannelService{}
			router := newChannelTestRouter(service)
			req := httptest.NewRequest(http.MethodPost, "/channels", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.False(t, service.createCalled, "invalid request must not reach the service")
		})
	}
}

func TestChannelAPI_Get(t *testing.T) {
	id := uuid.New()
	channel := &domain.Channel{Id: id, Name: "Test Channel", IsActive: true}
	tests := []struct {
		name       string
		path       string
		service    *fakeChannelService
		wantStatus int
		wantCall   bool
	}{
		{"success", "/channels/" + id.String(), &fakeChannelService{channel: channel}, http.StatusOK, true},
		{"invalid ID", "/channels/not-a-uuid", &fakeChannelService{}, http.StatusBadRequest, false},
		{"not found", "/channels/" + id.String(), &fakeChannelService{err: errors.New("not found")}, http.StatusNotFound, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newChannelTestRouter(tc.service)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))

			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, tc.wantCall, tc.service.getCalled)
			if tc.wantCall {
				assert.Equal(t, id, tc.service.requestedID)
			}
		})
	}
}

func TestChannelAPI_Update(t *testing.T) {
	id := uuid.New()
	service := &fakeChannelService{channel: &domain.Channel{
		Id:             id,
		Name:           "Updated Channel",
		IsActive:       true,
		FeePercent:     5,
		SuccessPercent: 80,
		LatencyMs:      300,
	}}
	router := newChannelTestRouter(service)
	req := httptest.NewRequest(
		http.MethodPut,
		"/channels/"+id.String(),
		strings.NewReader(`{"name":"Updated Channel","isActive":true,"feePercent":5,"successPercent":80,"latencyMs":300}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, service.updateCalled)
	assert.Equal(t, id, service.updatedInput.Id)
	assert.Equal(t, int64(80), service.updatedInput.SuccessPercent)
}
