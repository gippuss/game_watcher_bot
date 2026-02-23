package bot

import (
	"context"
	"fmt"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleList(c telebot.Context) error {
	ctx := context.Background()
	games, err := b.gamesQuery.Get(ctx, model.GameFilter{})
	if err != nil {
		return c.Send("Ошибка БД: " + err.Error())
	}
	if len(games) == 0 {
		return c.Send("Список игр пуст.")
	}

	gamesForUpdate, err := b.gamesQuery.GetGamesWithUpdates(ctx)
	if err != nil {
		return c.Send("Ошибка БД: " + err.Error())
	}

	return c.Send(fmt.Sprintf("В базе сейчас %d игр, требуется обновление у %d игр.", len(games), len(gamesForUpdate)))
}
