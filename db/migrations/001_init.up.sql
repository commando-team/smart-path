CREATE TYPE payment_status AS ENUM ('pending', 'success', 'failure', 'routing', 'processing', 'indeterminate');

CREATE TABLE IF NOT EXISTS payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    amount BIGINT NOT NULL,
    status payment_status NOT NULL DEFAULT 'pending',
    CONSTRAINT check_amount_non_negative CHECK (amount >= 0)
);