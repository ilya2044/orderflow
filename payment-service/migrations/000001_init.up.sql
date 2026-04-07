CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TYPE payment_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'refunded');
CREATE TYPE payment_method AS ENUM ('card', 'bank_transfer', 'sbp', 'yookassa');

CREATE TABLE IF NOT EXISTS payments (
    id          UUID           PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id    UUID           NOT NULL,
    user_id     VARCHAR(50)    NOT NULL,
    amount      DECIMAL(12,2)  NOT NULL,
    status      payment_status NOT NULL DEFAULT 'pending',
    method      payment_method NOT NULL,
    external_id VARCHAR(255),
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id  ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id   ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status    ON payments(status);
