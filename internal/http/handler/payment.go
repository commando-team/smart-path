package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/commando-team/smart-path/internal/logger"
	"github.com/commando-team/smart-path/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PaymentRequest struct {
	UserId         uuid.UUID `json:"userId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Amount         int64     `json:"amount" binding:"required,gt=0" example:"1000"`
	IdempotencyKey string    `json:"idempotencyKey,omitempty" binding:"omitempty,max=255" example:"pay-req-001"`
}

type PaymentResponse struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Status    string    `json:"status"`
}

type PaymentDetailsResponse struct {
	ID             uuid.UUID        `json:"ID"`
	UserID         uuid.UUID        `json:"UserID"`
	Amount         int64            `json:"Amount"`
	Status         db.PaymentStatus `json:"Status"`
	IdempotencyKey string           `json:"idempotencyKey,omitempty"`
}

// Pay godoc
// @Summary      Совершить платёж
// @Description  Регистрирует платёж и отправляет его на обработку (поддерживает Idempotency-Key)
// @Tags         Payment
// @Accept       json
// @Produce      json
// @Param        Idempotency-Key header string false "Ключ идемпотентности запроса"
// @Param        request body handler.PaymentRequest true "Данные платежа"
// @Success      201 {object} handler.PaymentResponse
// @Success      200 {object} handler.PaymentResponse
// @Router       /payment [post]
func (h *ApiHandler) Pay(c *gin.Context) {
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	headerKey := strings.TrimSpace(c.GetHeader("Idempotency-Key"))
	bodyKey := strings.TrimSpace(req.IdempotencyKey)

	if len(headerKey) > 255 {
		c.Status(http.StatusBadRequest)
		return
	}
	if headerKey != "" && bodyKey != "" && headerKey != bodyKey {
		c.Status(http.StatusBadRequest)
		return
	}

	idempotencyKey := headerKey
	if idempotencyKey == "" {
		idempotencyKey = bodyKey
	}

	result, err := h.payment.Pay(c.Request.Context(), req.UserId, req.Amount, idempotencyKey)
	if err != nil {
		if errors.Is(err, service.ErrIdempotencyConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		h.log.Error("payment handler error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	statusCode := http.StatusOK
	if result.Created {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, PaymentResponse{
		PaymentID: result.PaymentID,
		Status:    strings.ToUpper(string(result.Status)),
	})
}

// GetPayment godoc
// @Summary      Получить данные о платёже
// @Tags         Payment
// @Produce      json
// @Param        id path string true "Идентификатор платежа (UUID)"
// @Success      200 {object} handler.PaymentDetailsResponse
// @Router       /payment/{id} [get]
func (h *ApiHandler) GetPayment(c *gin.Context) {
	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	payment, err := h.payment.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Status(http.StatusNotFound)
			return
		}
		h.log.Error("failed to get payment", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	resp := PaymentDetailsResponse{
		ID:     payment.ID,
		UserID: payment.UserID,
		Amount: payment.Amount,
		Status: payment.Status,
	}
	if payment.IdempotencyKey.Valid {
		resp.IdempotencyKey = payment.IdempotencyKey.String
	}

	c.JSON(http.StatusOK, resp)
}
