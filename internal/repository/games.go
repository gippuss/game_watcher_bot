package repository

import (
	"context"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/gippuss/datagate"
	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GamesQuery interface {
	datagate.DataGate[model.Game, model.GameFilter]
	GetGamesWithUpdates(ctx context.Context) ([]model.Game, error)
	SyncAllGames(ctx context.Context) (int64, error)
}

type games struct {
	pool    *pgxpool.Pool
	builder squirrel.StatementBuilderType
	datagate.DataGate[model.Game, model.GameFilter]
}

func NewGamesQuery(pool *pgxpool.Pool) (GamesQuery, error) {
	dg, err := datagate.NewDataGate[model.Game, model.GameFilter](
		model.TableNameGames,
		"id",
		pool,
	)
	if err != nil {
		return nil, err
	}
	return &games{
		pool:     pool,
		builder:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		DataGate: dg,
	}, nil
}

func (r *games) GetGamesWithUpdates(ctx context.Context) ([]model.Game, error) {
	all, err := r.DataGate.Get(ctx, model.GameFilter{})
	if err != nil {
		return nil, err
	}
	var out []model.Game
	for _, g := range all {
		if g.HasUpdate() {
			out = append(out, g)
		}
	}
	return out, nil
}

func (r *games) SyncAllGames(ctx context.Context) (int64, error) {
	sql, args, err := r.builder.Update(model.TableNameGames).
		Set(model.TableGameColumnCurrentVersion, squirrel.Expr(model.TableGameColumnLatestVersion)).
		Where(squirrel.Expr(fmt.Sprintf("%s IS DISTINCT FROM %s", model.TableGameColumnCurrentVersion, model.TableGameColumnLatestVersion))).
		ToSql()
	if err != nil {
		return 0, err
	}

	rowsAffected, err := r.pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, err
	}

	return rowsAffected.RowsAffected(), nil
}
