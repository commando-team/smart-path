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
		resp.Body.Close()
		t.Fatalf("create payment: got HTTP %d, want 201", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()
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
		resp.Body.Close()
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
