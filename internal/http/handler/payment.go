package handler

import (
	"net/http"

	"github.com/commando-team/smart-path/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentRequest struct {
	UserId uuid.UUID `json:"userId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Amount int64     `json:"amount" binding:"required,gt=0" example:"1000"`
}

type PaymentResponse struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Status    string    `json:"status"`
}

// Pay godoc
// @Summary      Совершить платёж
// @Description  Регистрирует платёж и отправляет его на обработку
// @Tags         Payment
// @Accept       json
// @Produce      json
// @Param        request body handler.PaymentRequest true "Данные платежа"
// @Router       /payment [post]
func (h *ApiHandler) Pay(c *gin.Context) {
	var req PaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	paymentID, err := h.payment.Pay(c.Request.Context(), req.UserId, req.Amount)
	if err != nil {
		h.log.Error("payment handler error", logger.Err(err))
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusCreated, PaymentResponse{
		PaymentID: paymentID,
		Status:    "PENDING",
	})
}

// GetPayment godoc
// @Summary      Получить данные о платёже
// @Tags         Payment
// @Produce      json
// @Param        id path string  true "Идентификатор платежа (UUID)"
// @Success      200  {object}  db.Payment
// @Router       /payment/{id} [get]
func (h *ApiHandler) GetPayment(c *gin.Context) {
	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	payment, err := h.payment.GetPayment(c.Request.Context(), paymentID)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, payment)
}
