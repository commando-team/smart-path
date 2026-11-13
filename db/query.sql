-- name: CreatePayment :one
INSERT INTO payments (user_id, amount, status)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetPayment :one
SELECT * FROM payments
WHERE id = $1 LIMIT 1;

-- name: UpdatePaymentStatus :exec
UPDATE payments
SET status = $2
WHERE id = $1;

-- name: ListChannels :many
SELECT * FROM channels
ORDER BY name ASC;

-- name: ListActiveChannels :many
SELECT * FROM channels
WHERE is_active = true
ORDER BY name ASC;

-- name: GetChannelByID :one
SELECT * FROM channels
WHERE id = $1 LIMIT 1;

-- name: CreateChannel :one
INSERT INTO channels (name, is_active, fee_percent, success_percent, latency_ms)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateChannel :one
UPDATE channels
SET name = $2, is_active = $3, fee_percent = $4, success_percent = $5, latency_ms = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateChannelMetrics :exec
UPDATE channels
SET total_requests = $2,
    failed_requests = $3,
    consecutive_failures = $4,
    success_percent = $5,
    latency_ms = $6,
    is_active = $7,
    updated_at = NOW()
WHERE id = $1;