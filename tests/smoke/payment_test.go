//go:build smoke

package smoke

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestPaymentCreation checks that a payment can be created and read back.
func TestPaymentCreation(t *testing.T) {
	checkPayment(t, false)
}

// TestPaymentProcessing also checks asynchronous processing through Kafka.
func TestPaymentProcessing(t *testing.T) {
	checkPayment(t, true)
}

func TestPaymentIdempotency(t *testing.T) {
	baseURL := os.Getenv("SMOKE_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 15 * time.Second}

	userID := uuid.New()
	idemKey := "smoke-" + uuid.New().String()
	body := fmt.Sprintf(`{"userId":%q,"amount":1000}`, userID.String())

	doPay := func(payload string) (int, uuid.UUID) {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/payment", bytes.NewBufferString(payload))
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", idemKey)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("do request: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var res struct {
			PaymentID uuid.UUID `json:"paymentId"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&res)
		return resp.StatusCode, res.PaymentID
	}

	code1, id1 := doPay(body)
	if code1 != http.StatusCreated || id1 == uuid.Nil {
		t.Fatalf("first request: got status=%d id=%s, want 201", code1, id1)
	}

	code2, id2 := doPay(body)
	if code2 != http.StatusOK || id2 != id1 {
		t.Fatalf("duplicate request: got status=%d id=%s, want 200 id=%s", code2, id2, id1)
	}

	conflictBody := fmt.Sprintf(`{"userId":%q,"amount":2000}`, userID.String())
	code3, _ := doPay(conflictBody)
	if code3 != http.StatusConflict {
		t.Fatalf("conflict request: got status=%d, want 409", code3)
	}
}

func checkPayment(t *testing.T, waitForProcessing bool) {
	t.Helper()
	baseURL := os.Getenv("SMOKE_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	client := &http.Client{Timeout: 15 * time.Second}
	userID := uuid.New()
	body := fmt.Sprintf(`{"userId":%q,"amount":1000}`, userID.String())
	resp, err := client.Post(baseURL+"/payment", "application/json", bytes.NewBufferString(body))
	if err != nil {
		t.Fatalf("create payment: %v", err)
	}
	var created struct {
		PaymentID uuid.UUID `json:"paymentId"`
		Status    string    `json:"status"`
	}
	if resp.StatusCode != http.StatusCreated {
		_ = resp.Body.Close()
		t.Fatalf("create payment: got HTTP %d, want 201", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()
	if err != nil || created.PaymentID == uuid.Nil || created.Status != "PENDING" {
		t.Fatalf("unexpected creation response: %+v, error: %v", created, err)
	}

	deadline := time.Now().Add(30 * time.Second)
	for {
		resp, err := client.Get(baseURL + "/payment/" + created.PaymentID.String())
		if err != nil {
			t.Fatalf("get payment: %v", err)
		}
		var payment struct {
			ID     uuid.UUID
			UserID uuid.UUID
			Amount int64
			Status string
		}
		err = json.NewDecoder(resp.Body).Decode(&payment)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || err != nil {
			t.Fatalf("get payment: HTTP %d, decode error: %v", resp.StatusCode, err)
		}
		if payment.ID != created.PaymentID || payment.UserID != userID || payment.Amount != 1000 {
			t.Fatalf("unexpected saved payment: %+v", payment)
		}
		if !waitForProcessing {
			return
		}
		switch payment.Status {
		case "success", "failure":
			t.Logf("payment %s reached final status %s", payment.ID, payment.Status)
			return
		case "pending", "routing", "processing", "indeterminate":
		default:
			t.Fatalf("unknown payment status %q", payment.Status)
		}
		if time.Now().After(deadline) {
			t.Fatalf("payment did not reach a final status within 30 seconds; last status: %s", payment.Status)
		}
		time.Sleep(500 * time.Millisecond)
	}
}
