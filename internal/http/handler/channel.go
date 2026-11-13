package handler

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/commando-team/smart-path/internal/domain"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChannelService interface {
	ListChannels(ctx context.Context) ([]*domain.Channel, error)
	GetChannel(ctx context.Context, id uuid.UUID) (*domain.Channel, error)
	CreateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error)
	UpdateChannel(ctx context.Context, ch *domain.Channel) (*domain.Channel, error)
}

type ChannelHandler struct {
	log     *slog.Logger
	service ChannelService
}

func NewChannelHandler(log *slog.Logger, service ChannelService) *ChannelHandler {
	return &ChannelHandler{
		log:     log,
		service: service,
	}
}

type CreateChannelRequest struct {
	Name           string `json:"name" binding:"required"`
	IsActive       *bool  `json:"isActive" binding:"required"`
	FeePercent     int64  `json:"feePercent" binding:"gte=0"`
	SuccessPercent int64  `json:"successPercent" binding:"gte=0,lte=100"`
	LatencyMs      int64  `json:"latencyMs" binding:"gte=0"`
}

type UpdateChannelRequest struct {
	Name           string `json:"name" binding:"required"`
	IsActive       *bool  `json:"isActive" binding:"required"`
	FeePercent     int64  `json:"feePercent" binding:"gte=0"`
	SuccessPercent int64  `json:"successPercent" binding:"gte=0,lte=100"`
	LatencyMs      int64  `json:"latencyMs" binding:"gte=0"`
}

// ListChannels godoc
// @Summary      Список платёжных каналов
// @Description  Возвращает все каналы и их текущие метрики
// @Tags         Channel
// @Produce      json
// @Success      200 {array} domain.Channel
// @Router       /channels [get]
func (h *ChannelHandler) ListChannels(c *gin.Context) {
	channels, err := h.service.ListChannels(c.Request.Context())
	if err != nil {
		h.log.Error("failed to get channel list", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, channels)
}

// GetChannel godoc
// @Summary      Получить платёжный канал
// @Description  Возвращает информацию о канале по идентификатору
// @Tags         Channel
// @Produce      json
// @Param        id path string true "Идентификатор канала (UUID)"
// @Success      200 {object} domain.Channel
// @Router       /channels/{id} [get]
func (h *ChannelHandler) GetChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ch, err := h.service.GetChannel(c.Request.Context(), id)
	if err != nil {
		h.log.Error("failed to get channel", logger.Err(err))
		c.Status(http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, ch)
}

// CreateChannel godoc
// @Summary      Создать платёжный канал
// @Description  Добавляет новый платёжный канал в систему
// @Tags         Channel
// @Accept       json
// @Produce      json
// @Param        request body handler.CreateChannelRequest true "Данные нового канала"
// @Success      201 {object} domain.Channel
// @Router       /channels [post]
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ch := &domain.Channel{
		Name:           req.Name,
		IsActive:       *req.IsActive,
		FeePercent:     req.FeePercent,
		SuccessPercent: req.SuccessPercent,
		LatencyMs:      req.LatencyMs,
	}

	created, err := h.service.CreateChannel(c.Request.Context(), ch)
	if err != nil {
		h.log.Error("failed to create channel", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// UpdateChannel godoc
// @Summary      Обновить платёжный канал
// @Description  Обновляет параметры существующего канала
// @Tags         Channel
// @Accept       json
// @Produce      json
// @Param        id path string true "Идентификатор канала (UUID)"
// @Param        request body handler.UpdateChannelRequest true "Новые данные канала"
// @Success      200 {object} domain.Channel
// @Router       /channels/{id} [put]
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	var req UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	ch := &domain.Channel{
		Id:             id,
		Name:           req.Name,
		IsActive:       *req.IsActive,
		FeePercent:     req.FeePercent,
		SuccessPercent: req.SuccessPercent,
		LatencyMs:      req.LatencyMs,
	}

	updated, err := h.service.UpdateChannel(c.Request.Context(), ch)
	if err != nil {
		h.log.Error("failed to update channel", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, updated)
}
