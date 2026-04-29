package bot

import (
	"context"
	"fmt"
	"html"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/gippuss/game_watcher_bot/internal/parser"
	"github.com/samber/lo"
	"gopkg.in/telebot.v3"
)

func (b *bot) handleCheck(c telebot.Context) error {
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

	sem := make(chan struct{}, b.concurrency)
	var wg sync.WaitGroup
	var updatedCount atomic.Int64
	var sendMu sync.Mutex

	for i := range games {
		g := games[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			b.checkOneGame(ctx, c, g, &updatedCount, &sendMu)
		}()
	}

	wg.Wait()
	return c.Send(fmt.Sprintf("Проверено игр: %d. Необходимо обновить игр: %d.", len(games), updatedCount.Load()))
}

func (b *bot) checkOneGame(ctx context.Context, c telebot.Context, g model.Game, updatedCount *atomic.Int64, sendMu *sync.Mutex) {
	latest, err := parser.LatestVersion(ctx, g.WebsiteURL)
	if err != nil {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("Ошибка при получении актуальной версии игры %s: %v.", g.Name, err))
		sendMu.Unlock()
		slog.Error("failed to get latest version", "game", g.Name, "error", err)
		return
	}
	if latest == "" {
		slog.Error("failed to get latest version, empty", "game", g.Name)
		return
	}

	if !strings.EqualFold(g.CurrentVersion, latest) {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("<i>%s</i>\n<b>Установлено:</b> %s\n<b>Доступно:</b> %s\n%s\n\n",
			html.EscapeString(g.Name), html.EscapeString(g.CurrentVersion),
			html.EscapeString(latest), html.EscapeString(g.WebsiteURL)),
			&telebot.SendOptions{ParseMode: telebot.ModeHTML},
		)
		sendMu.Unlock()
		updatedCount.Add(1)
	}

	if strings.EqualFold(lo.FromPtr(g.LatestVersion), latest) {
		return
	}

	if err := b.gamesQuery.Update(ctx, model.GameFilter{Name: &g.Name}, map[string]interface{}{
		model.TableGameColumnLatestVersion: latest,
		model.TableGameColumnUpdatedAt:     time.Now(),
	}); err != nil {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("Ошибка при сохранении игры %s: %v.", g.Name, err))
		sendMu.Unlock()
		slog.Error("failed to update game", "game", g.Name, "error", err)
	}
}
