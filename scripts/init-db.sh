#!/bin/bash
set -e

create_db() {
  local db=$1
  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres -tc \
    "SELECT 1 FROM pg_database WHERE datname='$db'" | grep -q 1 \
    || psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres \
       -c "CREATE DATABASE $db"
  echo "  [ok] $db"
}

echo "==> Creating databases..."
create_db auth_db
create_db user_db
create_db order_db
create_db inventory_db
create_db payment_db

echo "==> Applying auth_db migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname auth_db <<-'EOSQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email         VARCHAR(255) UNIQUE NOT NULL,
    username      VARCHAR(100) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(500) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email               ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username            ON users(username);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id    ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);

INSERT INTO users (id, email, username, password_hash, role, is_active)
VALUES (
    uuid_generate_v4(),
    'admin@order.dev',
    'admin',
    '$2b$10$/G4JqZF7PdK.7iyn/4fAwOaDSvk5UtKma4uZ9mf.H9zubQolfUgEO',
    'admin',
    true
) ON CONFLICT (email) DO NOTHING;
EOSQL

echo "==> Applying user_db migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname user_db <<-'EOSQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS users (
    id         UUID         PRIMARY KEY,
    email      VARCHAR(255) UNIQUE NOT NULL,
    username   VARCHAR(100) UNIQUE NOT NULL,
    first_name VARCHAR(100) NOT NULL DEFAULT '',
    last_name  VARCHAR(100) NOT NULL DEFAULT '',
    phone      VARCHAR(20)  NOT NULL DEFAULT '',
    address    TEXT         NOT NULL DEFAULT '',
    avatar_url TEXT         NOT NULL DEFAULT '',
    role       VARCHAR(50)  NOT NULL DEFAULT 'user',
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_email    ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_role     ON users(role);
EOSQL

echo "==> Applying order_db migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname order_db <<-'EOSQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN
    CREATE TYPE order_status AS ENUM (
        'pending', 'confirmed', 'processing', 'shipped', 'delivered', 'cancelled', 'refunded'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS orders (
    id               UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id          UUID         NOT NULL,
    status           order_status NOT NULL DEFAULT 'pending',
    total_price      DECIMAL(12,2) NOT NULL,
    shipping_address TEXT         NOT NULL,
    notes            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS order_items (
    id           UUID         PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id     UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id   VARCHAR(50)  NOT NULL,
    product_name VARCHAR(255) NOT NULL,
    price        DECIMAL(10,2) NOT NULL,
    quantity     INTEGER      NOT NULL CHECK (quantity > 0),
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_orders_user_id       ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_orders_status        ON orders(status);
CREATE INDEX IF NOT EXISTS idx_orders_created_at    ON orders(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);

CREATE OR REPLACE FUNCTION update_order_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_orders_updated_at ON orders;
CREATE TRIGGER trigger_orders_updated_at
    BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_order_updated_at();
EOSQL

echo "==> Applying inventory_db migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname inventory_db <<-'EOSQL'
CREATE TABLE IF NOT EXISTS inventory (
    product_id   VARCHAR(50)  PRIMARY KEY,
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    stock        INTEGER      NOT NULL DEFAULT 0 CHECK (stock >= 0),
    reserved     INTEGER      NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_stock ON inventory(stock);
EOSQL

echo "==> Applying payment_db migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname payment_db <<-'EOSQL'
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

DO $$ BEGIN
    CREATE TYPE payment_status AS ENUM ('pending', 'processing', 'completed', 'failed', 'refunded');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE payment_method AS ENUM ('card', 'bank_transfer', 'sbp', 'yookassa');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

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

CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_user_id  ON payments(user_id);
CREATE INDEX IF NOT EXISTS idx_payments_status   ON payments(status);
EOSQL

echo "==> All databases and migrations applied successfully!"
echo "==> Admin credentials: admin@order.dev / admin123"
