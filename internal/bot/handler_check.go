package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/gippuss/game_watcher_bot/internal/parser"
	"github.com/samber/lo"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleCheck(c telebot.Context) error {
	ctx := context.Background()
	games, err := b.gamesQuery.Get(ctx, model.GameFilter{})
	if err != nil {
		return c.Send("Ошибка БД: " + err.Error())
	}
	if len(games) == 0 {
		return c.Send("Список игр пуст. Сначала загрузите CSV через /upload.")
	}

	if err := c.Send("Проверяю обновления. Игры, которые нужно обновить выведутся ниже."); err != nil {
		return err
	}
	updated := 0
	for _, g := range games {
		latest, err := parser.LatestVersion(ctx, g.WebsiteURL)
		if err != nil {
			_ = c.Send(fmt.Sprintf("Ошибка при получении актуальной версии игры %s: %v.", g.Name, err))
			slog.Error("failed to get latest version", "error", err)
			continue
		}
		if latest == "" {
			slog.Error("failed to get latest version, empty")
			continue
		}

		if !strings.EqualFold(g.CurrentVersion, latest) {
			_ = c.Send(fmt.Sprintf("<i>%s</i>\n<b>Установлено:</b> %s\n<b>Доступно:</b> %s\n%s\n\n",
				html.EscapeString(g.Name), html.EscapeString(g.CurrentVersion),
				html.EscapeString(*g.LatestVersion), html.EscapeString(g.WebsiteURL)),
				&telebot.SendOptions{ParseMode: telebot.ModeHTML},
			)
			updated++
		}

		if strings.EqualFold(lo.FromPtr(g.LatestVersion), latest) {
			continue
		}

		if err := b.gamesQuery.Update(ctx, model.GameFilter{Name: &g.Name}, map[string]interface{}{
			model.TableGameColumnLatestVersion: latest,
			model.TableGameColumnUpdatedAt:     time.Now(),
		}); err != nil {
			_ = c.Send(fmt.Sprintf("Ошибка при сохранении игры %s: %v.", g.Name, err))
			continue
		}
	}
	return c.Send(fmt.Sprintf("Проверено игр: %d. Необходимо обновить игр: %d.", len(games), updated))
}
