package handler

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/commando-team/smart-path/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type fakePaymentService struct {
	paymentID uuid.UUID
}

func (f *fakePaymentService) Pay(
	ctx context.Context,
	userID uuid.UUID,
	amount int64,
) (uuid.UUID, error) {
	return f.paymentID, nil
}

func (f *fakePaymentService) GetPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) (*db.Payment, error) {
	return nil, nil
}

func TestPayAPI_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedPaymentID := uuid.New()
	service := &fakePaymentService{
		paymentID: expectedPaymentID,
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiHandler := NewApiHandler(log, service)

	router := gin.New()
	router.POST("/payment", apiHandler.Pay)

	userID := uuid.New()
	body := `{"userId":"` + userID.String() + `","amount":1000}`

	req := httptest.NewRequest(
		http.MethodPost,
		"/payment",
		strings.NewReader(body),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), expectedPaymentID.String())
	assert.Contains(t, rec.Body.String(), `"status":"PENDING"`)
}

func TestPayAPI_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &fakePaymentService{
		paymentID: uuid.New(),
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	apiHandler := NewApiHandler(log, service)

	router := gin.New()
	router.POST("/payment", apiHandler.Pay)

	req := httptest.NewRequest(
		http.MethodPost,
		"/payment",
		strings.NewReader(`{"amount":`),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
