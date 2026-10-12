package domain

import "github.com/google/uuid"

type Channel struct {
	Id             uuid.UUID
	Name           string
	IsActive       bool
	FeePercent     int64
	SuccessPercent int64
	LatencyMs      int64
}
