package bot

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleUpdated(c telebot.Context) error {
	payload := strings.TrimSpace(c.Message().Payload)
	if payload == "" {
		return c.Send("Использование: /updated <название1>, <название2>, ... — игры, которые вы уже обновили. Названия с запятой — в кавычках: \"Игра, часть 2\".")
	}
	ctx := context.Background()

	if strings.EqualFold(payload, "all") {
		count, err := b.gamesQuery.SyncAllGames(ctx)
		if err != nil {
			_ = c.Send("Ошибка БД: " + err.Error())
			return nil
		}
		return c.Send(fmt.Sprintf("Обновлено игр: %d", count))
	}

	names := splitNames(payload)
	updated := 0
	for _, name := range names {
		if name == "" {
			continue
		}
		games, err := b.gamesQuery.Get(ctx, model.GameFilter{Name: &name})
		if err != nil {
			_ = c.Send(fmt.Sprintf("%q: ошибка БД — %v", name, err))
			continue
		}
		if len(games) == 0 {
			_ = c.Send(fmt.Sprintf("Игра %q не найдена.", name))
			continue
		}
		g := games[0]
		if g.LatestVersion == nil || *g.LatestVersion == "" {
			_ = c.Send(fmt.Sprintf("%q: нет актуальной версии. Сначала выполните /check.", name))
			continue
		}
		err = b.gamesQuery.Update(ctx, model.GameFilter{Name: &name}, map[string]interface{}{
			model.TableGameColumnCurrentVersion: *g.LatestVersion,
			model.TableGameColumnUpdatedAt:      time.Now(),
		})
		if err != nil {
			_ = c.Send(fmt.Sprintf("%q: не удалось обновить — %v", name, err))
			continue
		}
		updated++
	}
	return c.Send(fmt.Sprintf("Отмечено как обновлённые: %d из %d.", updated, len(names)))
}
