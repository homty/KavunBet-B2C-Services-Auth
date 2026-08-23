-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_pkey;

ALTER TABLE user_wallets 
ADD COLUMN id BIGSERIAL PRIMARY KEY,
ADD CONSTRAINT user_wallets_user_id_key UNIQUE (user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_user_id_key;
ALTER TABLE user_wallets DROP CONSTRAINT IF EXISTS user_wallets_pkey;
ALTER TABLE user_wallets DROP COLUMN IF EXISTS id;
ALTER TABLE user_wallets ADD CONSTRAINT user_wallets_pkey PRIMARY KEY (user_id);
-- +goose StatementEnd
