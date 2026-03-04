-- +goose NO TRANSACTION
-- +goose Up
DROP INDEX CONCURRENTLY IF EXISTS idx_games_name;
CREATE UNIQUE INDEX CONCURRENTLY idx_games_name_unique ON games(name);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_games_name_unique;
CREATE INDEX CONCURRENTLY idx_games_name ON games(name);
