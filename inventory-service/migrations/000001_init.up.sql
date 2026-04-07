CREATE TABLE IF NOT EXISTS inventory (
    product_id   VARCHAR(50)  PRIMARY KEY,
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    stock        INTEGER      NOT NULL DEFAULT 0 CHECK (stock >= 0),
    reserved     INTEGER      NOT NULL DEFAULT 0 CHECK (reserved >= 0),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_inventory_stock ON inventory(stock);
