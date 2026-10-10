-- name: CreatePayment :one
INSERT INTO payments (user_id, amount, status)
VALUES ($1, $2, $3)
RETURNING id;

-- name: GetPayment :one
SELECT * FROM payments
WHERE id = $1 LIMIT 1;