-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS games (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    website_url TEXT NOT NULL,
    current_version TEXT NOT NULL,
    latest_version TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_games_name ON games(name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_games_name;
DROP TABLE IF EXISTS games;
-- +goose StatementEnd
