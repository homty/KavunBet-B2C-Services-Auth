-- +goose Up
-- +goose StatementBegin

-- Passwords are deliberately not stored in this migration. Docker bootstrap
-- must assign passwords before applications connect with these login roles.
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

CREATE SCHEMA auth_service;
CREATE SCHEMA user_service;
CREATE SCHEMA wallet_service;

REVOKE CREATE ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON SCHEMA auth_service FROM PUBLIC;
REVOKE ALL ON SCHEMA user_service FROM PUBLIC;
REVOKE ALL ON SCHEMA wallet_service FROM PUBLIC;

DO $$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM PUBLIC', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO auth_app', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO user_app', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO wallet_app', current_database());
END
$$;

-- These foreign keys cross service ownership boundaries. Registration cleanup
-- is coordinated through gRPC instead of database cascades.
ALTER TABLE public.user_profiles
DROP CONSTRAINT user_profiles_user_id_fkey;

ALTER TABLE public.user_wallets
DROP CONSTRAINT user_wallets_user_id_fkey;

ALTER TABLE public.users SET SCHEMA auth_service;
ALTER TABLE public.user_profiles SET SCHEMA user_service;
ALTER TABLE public.user_wallets SET SCHEMA wallet_service;

ALTER TABLE user_service.user_profiles RENAME TO profiles;
ALTER TABLE wallet_service.user_wallets RENAME TO wallets;

GRANT USAGE ON SCHEMA auth_service TO auth_app;
GRANT USAGE ON SCHEMA user_service TO user_app;
GRANT USAGE ON SCHEMA wallet_service TO wallet_app;

GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA auth_service TO auth_app;
GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA user_service TO user_app;
GRANT SELECT, INSERT, UPDATE, DELETE
ON ALL TABLES IN SCHEMA wallet_service TO wallet_app;

GRANT USAGE, SELECT, UPDATE
ON ALL SEQUENCES IN SCHEMA auth_service TO auth_app;
GRANT USAGE, SELECT, UPDATE
ON ALL SEQUENCES IN SCHEMA user_service TO user_app;
GRANT USAGE, SELECT, UPDATE
ON ALL SEQUENCES IN SCHEMA wallet_service TO wallet_app;

-- Future objects created by the migration account receive the same runtime
-- permissions automatically.
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_service
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO auth_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA user_service
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO user_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA wallet_service
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO wallet_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA auth_service
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO auth_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA user_service
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO user_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA wallet_service
GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO wallet_app;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin

ALTER DEFAULT PRIVILEGES IN SCHEMA auth_service
REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM auth_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA user_service
REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM user_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA wallet_service
REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM wallet_app;

ALTER DEFAULT PRIVILEGES IN SCHEMA auth_service
REVOKE USAGE, SELECT, UPDATE ON SEQUENCES FROM auth_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA user_service
REVOKE USAGE, SELECT, UPDATE ON SEQUENCES FROM user_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA wallet_service
REVOKE USAGE, SELECT, UPDATE ON SEQUENCES FROM wallet_app;

ALTER TABLE user_service.profiles RENAME TO user_profiles;
ALTER TABLE wallet_service.wallets RENAME TO user_wallets;

ALTER TABLE auth_service.users SET SCHEMA public;
ALTER TABLE user_service.user_profiles SET SCHEMA public;
ALTER TABLE wallet_service.user_wallets SET SCHEMA public;

ALTER TABLE public.user_profiles
ADD CONSTRAINT user_profiles_user_id_fkey
FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

ALTER TABLE public.user_wallets
ADD CONSTRAINT user_wallets_user_id_fkey
FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;

REVOKE ALL PRIVILEGES ON TABLE public.users FROM auth_app;
REVOKE ALL PRIVILEGES ON TABLE public.user_profiles FROM user_app;
REVOKE ALL PRIVILEGES ON TABLE public.user_wallets FROM wallet_app;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public
FROM auth_app, user_app, wallet_app;

DROP SCHEMA auth_service;
DROP SCHEMA user_service;
DROP SCHEMA wallet_service;

DO $$
BEGIN
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM auth_app', current_database());
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM user_app', current_database());
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM wallet_app', current_database());
    EXECUTE format('GRANT CONNECT ON DATABASE %I TO PUBLIC', current_database());
END
$$;

DROP ROLE auth_app;
DROP ROLE user_app;
DROP ROLE wallet_app;

GRANT CREATE ON SCHEMA public TO PUBLIC;

-- +goose StatementEnd
