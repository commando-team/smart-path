package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
)

type createPaymentResponse struct {
	PaymentID uuid.UUID `json:"paymentId"`
	Status    string    `json:"status"`
}

type paymentStatusResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"userID"`
	Amount int64     `json:"amount"`
	Status string    `json:"status"`
}

type channelResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	IsActive            bool      `json:"isActive"`
	FeePercent          int64     `json:"feePercent"`
	SuccessPercent      int64     `json:"successPercent"`
	LatencyMs           int64     `json:"latencyMs"`
	TotalRequests       int64     `json:"totalRequests"`
	FailedRequests      int64     `json:"failedRequests"`
	ConsecutiveFailures int64     `json:"consecutiveFailures"`
}

func main() {
	baseURL := flag.String("url", "http://localhost:8080", "server base URL")
	totalRequests := flag.Int("n", 50, "total payments count")
	concurrency := flag.Int("c", 5, "concurrent workers count")
	amount := flag.Int64("amount", 1000, "payment amount")
	flag.Parse()

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Info("starting simulation",
		slog.String("url", *baseURL),
		slog.Int("requests", *totalRequests),
		slog.Int("concurrency", *concurrency),
		slog.Int64("amount", *amount),
	)

	client := &http.Client{Timeout: 15 * time.Second}

	var successfulPayments atomic.Int64
	var failedPayments atomic.Int64
	var totalErrors atomic.Int64

	jobs := make(chan int, *totalRequests)
	for i := 1; i <= *totalRequests; i++ {
		jobs <- i
	}
	close(jobs)

	var wg sync.WaitGroup
	startTime := time.Now()

	for workerID := 1; workerID <= *concurrency; workerID++ {
		wg.Add(1)
		go func(wID int) {
			defer wg.Done()

			for job := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				paymentID, err := sendPayment(ctx, client, *baseURL, *amount)
				if err != nil {
					totalErrors.Add(1)
					log.Error("failed to initiate payment", slog.Int("job", job), slog.String("error", err.Error()))
					continue
				}

				status, err := pollPaymentStatus(ctx, client, *baseURL, paymentID)
				if err != nil {
					totalErrors.Add(1)
					log.Error("failed during polling payment", slog.String("payment_id", paymentID.String()), slog.String("error", err.Error()))
					continue
				}

				if status == "success" {
					successfulPayments.Add(1)
				} else {
					failedPayments.Add(1)
				}

				log.Info("payment finished",
					slog.Int("job", job),
					slog.String("payment_id", paymentID.String()),
					slog.String("final_status", status),
				)
			}
		}(workerID)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	log.Info("simulation summary",
		slog.Duration("duration", elapsed),
		slog.Int64("success", successfulPayments.Load()),
		slog.Int64("failure", failedPayments.Load()),
		slog.Int64("network_errors", totalErrors.Load()),
		slog.Float64("requests_per_second", float64(*totalRequests)/elapsed.Seconds()),
	)

	printChannelStats(ctx, client, *baseURL, log)
}

func sendPayment(ctx context.Context, client *http.Client, baseURL string, amount int64) (uuid.UUID, error) {
	reqBody, _ := json.Marshal(map[string]any{
		"userId": uuid.New().String(),
		"amount": amount,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/payment", bytes.NewReader(reqBody))
	if err != nil {
		return uuid.Nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return uuid.Nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return uuid.Nil, fmt.Errorf("bad status %d: %s", resp.StatusCode, string(body))
	}

	var data createPaymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return uuid.Nil, err
	}
	return data.PaymentID, nil
}

func pollPaymentStatus(ctx context.Context, client *http.Client, baseURL string, paymentID uuid.UUID) (string, error) {
	timeout := time.After(45 * time.Second)
	ticker := time.NewTicker(300 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout:
			return "", fmt.Errorf("timeout waiting for payment %s", paymentID)
		case <-ticker.C:
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/payment/%s", baseURL, paymentID.String()), nil)
			if err != nil {
				return "", err
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			if resp.StatusCode != http.StatusOK {
				resp.Body.Close()
				continue
			}

			var statusResp paymentStatusResponse
			err = json.NewDecoder(resp.Body).Decode(&statusResp)
			resp.Body.Close()
			if err != nil {
				continue
			}

			if statusResp.Status == "success" || statusResp.Status == "failure" {
				return statusResp.Status, nil
			}
		}
	}
}

func printChannelStats(ctx context.Context, client *http.Client, baseURL string, log *slog.Logger) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/channels", nil)
	if err != nil {
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error("failed to retrieve channel stats", slog.String("error", err.Error()))
		return
	}
	defer resp.Body.Close()

	var channels []channelResponse
	if err := json.NewDecoder(resp.Body).Decode(&channels); err != nil {
		log.Error("failed to parse channels", slog.String("error", err.Error()))
		return
	}

	for _, ch := range channels {
		log.Info("channel metrics",
			slog.String("name", ch.Name),
			slog.Bool("active", ch.IsActive),
			slog.Int64("success_percent", ch.SuccessPercent),
			slog.Int64("latency_ms", ch.LatencyMs),
			slog.Int64("total_requests", ch.TotalRequests),
			slog.Int64("failed_requests", ch.FailedRequests),
			slog.Int64("consecutive_failures", ch.ConsecutiveFailures),
		)
	}
}
