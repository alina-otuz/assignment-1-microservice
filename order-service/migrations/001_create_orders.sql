-- migrations/001_create_orders.sql
-- Run against the 'orders' database (separate from payments DB).

CREATE TABLE IF NOT EXISTS orders (
    id              VARCHAR(36)  PRIMARY KEY,
    customer_id     VARCHAR(255) NOT NULL,
    item_name       VARCHAR(255) NOT NULL,
    amount          BIGINT       NOT NULL CHECK (amount > 0),  -- stored in cents; int64 only
    status          VARCHAR(50)  NOT NULL DEFAULT 'Pending',   -- Pending | Paid | Failed | Cancelled
    idempotency_key VARCHAR(255) UNIQUE,                       -- NULL when no key provided (bonus feature)
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_customer_id ON orders (customer_id);
CREATE INDEX IF NOT EXISTS idx_orders_status      ON orders (status);
