-- +goose Up
CREATE TABLE IF NOT EXISTS withdrawals (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    order_number VARCHAR(255) NOT NULL,
    sum DECIMAL(10,2) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX IF NOT EXISTS idx_withdrawals_processed_at ON withdrawals(processed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_withdrawals_processed_at;
DROP INDEX IF EXISTS idx_withdrawals_user_id;
DROP TABLE IF EXISTS withdrawals;