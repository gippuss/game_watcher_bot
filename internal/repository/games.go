package repository

import (
	"context"

	"github.com/gippuss/datagate"
	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GamesQuery interface {
	datagate.DataGate[model.Game, model.GameFilter]
	GetGamesWithUpdates(ctx context.Context) ([]model.Game, error)
}

type games struct {
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
	return &games{DataGate: dg}, nil
}

func (r games) GetGamesWithUpdates(ctx context.Context) ([]model.Game, error) {
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
