-- migrations/001_create_payments.sql
-- Run against the 'payments' database (completely separate from orders DB).

CREATE TABLE IF NOT EXISTS payments (
    id             VARCHAR(36)  PRIMARY KEY,
    order_id       VARCHAR(36)  NOT NULL,
    transaction_id VARCHAR(36)  NOT NULL,
    amount         BIGINT       NOT NULL CHECK (amount > 0),  -- stored in cents; int64 only
    status         VARCHAR(50)  NOT NULL                       -- Authorized | Declined
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments (order_id);
