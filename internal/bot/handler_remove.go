package bot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gippuss/datagate"
	"github.com/gippuss/game_watcher_bot/internal/model"
	"gopkg.in/telebot.v3"
)

func (b *bot) handleRemove(c telebot.Context) error {
	payload := strings.TrimSpace(c.Message().Payload)
	if payload == "" {
		return c.Send("Использование: /remove <название1>, <название2>, ... — игры, которые нужно удалить.")
	}

	ctx := context.Background()

	names := splitNames(payload)
	updated := 0
	for _, name := range names {
		if name == "" {
			continue
		}

		err := b.gamesQuery.Delete(ctx, model.GameFilter{Name: &name})
		if err != nil {
			if errors.Is(err, datagate.ErrNoRowsAffected) {
				return c.Send(fmt.Sprintf("Игра с названием %s не найдена.", name))
			}
			return c.Send("Не удалось удалить игру %s: %v", name, err.Error())
		}
		updated++
	}
	return c.Send(fmt.Sprintf("Успешно удалены: %d из %d.", updated, len(names)))
}
