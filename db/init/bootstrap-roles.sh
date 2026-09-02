#!/usr/bin/env bash
set -Eeuo pipefail

: "${PGHOST:?PGHOST is required}"
: "${PGPORT:?PGPORT is required}"
: "${PGDATABASE:?PGDATABASE is required}"
: "${PGUSER:?PGUSER is required}"
: "${PGPASSWORD:?PGPASSWORD is required}"
: "${AUTH_DB_PASSWORD:?AUTH_DB_PASSWORD is required}"
: "${USER_DB_PASSWORD:?USER_DB_PASSWORD is required}"
: "${WALLET_DB_PASSWORD:?WALLET_DB_PASSWORD is required}"

psql --set=ON_ERROR_STOP=1 \
    --set=auth_password="$AUTH_DB_PASSWORD" \
    --set=user_password="$USER_DB_PASSWORD" \
    --set=wallet_password="$WALLET_DB_PASSWORD" <<'SQL'
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_app') THEN
        CREATE ROLE auth_app LOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'user_app') THEN
        CREATE ROLE user_app LOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'wallet_app') THEN
        CREATE ROLE wallet_app LOGIN;
    END IF;
END
$$;

ALTER ROLE auth_app
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION
    PASSWORD :'auth_password';
ALTER ROLE user_app
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION
    PASSWORD :'user_password';
ALTER ROLE wallet_app
    NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION
    PASSWORD :'wallet_password';
SQL

echo "Database service roles are ready."
