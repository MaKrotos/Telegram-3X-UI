-- +goose Up
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    telegram_payment_charge_id VARCHAR(128) NOT NULL,
    telegram_user_id BIGINT NOT NULL,
    amount INTEGER NOT NULL,
    invoice_payload VARCHAR(255),
    status VARCHAR(32) NOT NULL,
    type VARCHAR(16) NOT NULL, -- 'payment' или 'refund'
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE IF EXISTS transactions; 