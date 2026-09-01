-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
ALTER COLUMN status SET DEFAULT 'PENDING';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users
ALTER COLUMN status SET DEFAULT 'ACTIVE';
-- +goose StatementEnd
