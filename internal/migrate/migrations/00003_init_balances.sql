-- +goose Up
CREATE TABLE IF NOT EXISTS balances (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    current DECIMAL(10,2) NOT NULL DEFAULT 0,
    withdrawn DECIMAL(10,2) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);

-- +goose Down
DROP INDEX IF EXISTS idx_balances_user_id;
DROP TABLE IF EXISTS balances;