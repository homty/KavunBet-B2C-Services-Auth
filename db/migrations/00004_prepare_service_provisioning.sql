-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ADD COLUMN status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
ADD CONSTRAINT users_status_check CHECK (status IN ('PENDING', 'ACTIVE', 'FAILED'));

ALTER TABLE user_profiles
ADD COLUMN username VARCHAR(50);

UPDATE user_profiles AS profile
SET username = users.username
FROM users
WHERE users.id = profile.user_id;

ALTER TABLE user_profiles
ALTER COLUMN username SET NOT NULL,
ADD CONSTRAINT user_profiles_username_key UNIQUE (username);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE user_profiles
DROP CONSTRAINT IF EXISTS user_profiles_username_key,
DROP COLUMN IF EXISTS username;

ALTER TABLE users
DROP CONSTRAINT IF EXISTS users_status_check,
DROP COLUMN IF EXISTS status;
-- +goose StatementEnd
