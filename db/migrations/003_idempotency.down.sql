ALTER TABLE payments
    DROP CONSTRAINT IF EXISTS uq_payments_idempotency_key,
    DROP COLUMN IF EXISTS idempotency_key;