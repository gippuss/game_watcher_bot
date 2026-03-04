-- +goose Up
-- +goose StatementBegin
ALTER TABLE games ADD CONSTRAINT uq_games_name UNIQUE USING INDEX idx_games_name_unique;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE games DROP CONSTRAINT uq_games_name;
-- +goose StatementEnd
