-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_profiles
ADD COLUMN display_name VARCHAR(100),
ADD COLUMN bio TEXT,
ADD COLUMN avatar_url VARCHAR(512);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_profiles
DROP COLUMN IF EXISTS avatar_url,
DROP COLUMN IF EXISTS bio,
DROP COLUMN IF EXISTS display_name;
-- +goose StatementEnd
