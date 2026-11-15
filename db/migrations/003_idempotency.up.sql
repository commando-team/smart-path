ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);

ALTER TABLE payments
    ADD CONSTRAINT uq_payments_idempotency_key UNIQUE (idempotency_key);    