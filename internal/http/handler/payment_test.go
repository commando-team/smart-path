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

	"github.com/commando-team/smart-path/internal/db"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Заглушка вместо настоящего сервиса платежей.
type fakePaymentService struct {
	paymentID uuid.UUID
	payment   *db.Payment
	err       error
	payCalled bool
	getCalled bool
	gotID     uuid.UUID
}

func (f *fakePaymentService) Pay(
	ctx context.Context,
	userID uuid.UUID,
	amount int64,
) (uuid.UUID, error) {
	f.payCalled = true
	return f.paymentID, f.err
}

func (f *fakePaymentService) GetPayment(
	ctx context.Context,
	paymentID uuid.UUID,
) (*db.Payment, error) {
	f.getCalled = true
	f.gotID = paymentID
	return f.payment, f.err
}

func newTestRouter(service *fakePaymentService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewApiHandler(log, service)
	router := gin.New()
	router.POST("/payment", h.Pay)
	router.GET("/payment/:id", h.GetPayment)
	return router
}

func TestPayAPI_InvalidFields(t *testing.T) {
	userID := uuid.New().String()
	tests := []struct {
		name string
		body string
	}{
		{"missing user ID", `{"amount":1000}`},
		{"invalid user ID", `{"userId":"not-a-uuid","amount":1000}`},
		{"missing amount", `{"userId":"` + userID + `"}`},
		{"zero amount", `{"userId":"` + userID + `","amount":0}`},
		{"negative amount", `{"userId":"` + userID + `","amount":-1}`},
		{"amount is a string", `{"userId":"` + userID + `","amount":"1000"}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakePaymentService{}
			router := newTestRouter(service)
			req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.False(t, service.payCalled, "invalid request must not reach the service")
		})
	}
}

func TestPayAPI_ServiceError(t *testing.T) {
	service := &fakePaymentService{err: errors.New("service failed")}
	router := newTestRouter(service)
	body := `{"userId":"` + uuid.New().String() + `","amount":1000}`
	req := httptest.NewRequest(http.MethodPost, "/payment", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.True(t, service.payCalled)
}

func TestGetPaymentAPI(t *testing.T) {
	id := uuid.New()
	payment := &db.Payment{ID: id, UserID: uuid.New(), Amount: 1000, Status: db.PaymentStatusPending}
	tests := []struct {
		name       string
		path       string
		service    *fakePaymentService
		wantStatus int
		wantCall   bool
	}{
		{"success", "/payment/" + id.String(), &fakePaymentService{payment: payment}, http.StatusOK, true},
		{"invalid ID", "/payment/not-a-uuid", &fakePaymentService{}, http.StatusBadRequest, false},
		{"service error", "/payment/" + id.String(), &fakePaymentService{err: errors.New("service failed")}, http.StatusInternalServerError, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			router := newTestRouter(tc.service)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			assert.Equal(t, tc.wantStatus, rec.Code)
			assert.Equal(t, tc.wantCall, tc.service.getCalled)
			if tc.wantCall {
				assert.Equal(t, id, tc.service.gotID)
			}
			if tc.wantStatus == http.StatusOK {
				assert.JSONEq(t, `{"ID":"`+id.String()+`","UserID":"`+payment.UserID.String()+`","Amount":1000,"Status":"pending"}`, rec.Body.String())
			}
		})
	}
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
