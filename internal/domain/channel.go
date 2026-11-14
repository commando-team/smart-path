package domain

import "github.com/google/uuid"

type Channel struct {
	Id                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	IsActive            bool      `json:"isActive"`
	FeePercent          int64     `json:"feePercent"`
	SuccessPercent      int64     `json:"successPercent"`
	LatencyMs           int64     `json:"latencyMs"`
	TotalRequests       int64     `json:"totalRequests"`
	FailedRequests      int64     `json:"failedRequests"`
	ConsecutiveFailures int64     `json:"consecutiveFailures"`
}
