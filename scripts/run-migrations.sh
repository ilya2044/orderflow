#!/bin/sh
set -e

export PGPASSWORD=postgres
export PGHOST=postgres
export PGUSER=postgres

echo "==> Waiting for postgres to be ready..."
until psql -d postgres -c '\q' 2>/dev/null; do
  echo "    postgres not ready yet, retrying in 2s..."
  sleep 2
done
echo "    postgres is ready!"

create_db() {
  local db=$1
  psql -d postgres -tc "SELECT 1 FROM pg_database WHERE datname='$db'" | grep -q 1 \
    && echo "  $db already exists" \
    || (psql -d postgres -c "CREATE DATABASE $db" && echo "  $db created")
}

echo "==> Creating databases..."
create_db auth_db
create_db user_db
create_db order_db
create_db inventory_db
create_db payment_db

echo "==> Running migrations..."
psql -d auth_db      -f /migrations/auth/000001_init.up.sql      && echo "  auth_db done"
psql -d user_db      -f /migrations/user/000001_init.up.sql      && echo "  user_db done"
psql -d order_db     -f /migrations/order/000001_init.up.sql     && echo "  order_db done"
psql -d inventory_db -f /migrations/inventory/000001_init.up.sql && echo "  inventory_db done"
psql -d payment_db   -f /migrations/payment/000001_init.up.sql   && echo "  payment_db done"

echo "==> All migrations completed!"
