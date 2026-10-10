package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PaymentRequest struct {
	UserId uuid.UUID `json:"userId" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Amount int64     `json:"amount" binding:"required,gt=0" example:"1000"`
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

	c.Status(http.StatusNotImplemented)
}
