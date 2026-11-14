CREATE TABLE IF NOT EXISTS channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    fee_percent BIGINT NOT NULL DEFAULT 0,
    success_percent BIGINT NOT NULL DEFAULT 100,
    latency_ms BIGINT NOT NULL DEFAULT 100,
    total_requests BIGINT NOT NULL DEFAULT 0,
    failed_requests BIGINT NOT NULL DEFAULT 0,
    consecutive_failures BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

INSERT INTO channels (name, is_active, fee_percent, success_percent, latency_ms) VALUES
('T-Bank', true, 0, 99, 100),
('Sberbank', true, 10, 80, 500),
('Alfa-Bank', true, 3, 90, 200)
ON CONFLICT DO NOTHING;