package bot

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"html"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/gippuss/datagate"
	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/gippuss/game_watcher_bot/internal/parser"
	"github.com/samber/lo"
	"gopkg.in/telebot.v3"
)

func (b *Bot) handleStart(c telebot.Context) error {
	text := `Привет! Я бот для отслеживания обновлений игр.

Команды:
/start - это сообщение
/upload - загрузить CSV (ссылка на игру, установленная версия)
/check - получисть список игр с устаревшими версиями и обновить бд
/list - количество игр
/updated <название1, название2, ...> - отметить игры как обновлённые (установленная версия = актуальная)
/remove <название1, название2, ...> - удалить игры`
	return c.Send(text)
}

func (b *Bot) handleUpload(c telebot.Context) error {
	doc := c.Message().Document
	if doc == nil {
		return c.Send("Отправьте файл (документ) с расширением .csv после команды /upload.")
	}
	if !strings.HasSuffix(strings.ToLower(doc.FileName), ".csv") {
		return c.Send("Нужен именно CSV файл.")
	}

	fileReader, err := b.tg.File(doc.MediaFile())
	if err != nil {
		return c.Send("Не удалось получить файл: " + err.Error())
	}
	defer fileReader.Close()
	data, err := io.ReadAll(fileReader)
	if err != nil {
		return c.Send("Не удалось скачать файл: " + err.Error())
	}
	csvReader := csv.NewReader(bytes.NewReader(data))
	csvReader.Comma = ','
	csvReader.TrimLeadingSpace = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return c.Send("Ошибка разбора CSV: " + err.Error())
	}
	_ = c.Send("Файл успешно скачан, приступаю к обработке.")

	ctx := context.Background()
	added := 0
	for i, row := range records {
		if len(row) < 2 {
			_ = c.Send(fmt.Sprintf("Строка %d: нужны две колонки (ссылка, установленная версия).", i+1))
			continue
		}
		gameURL := strings.TrimSpace(row[0])
		currentVer := strings.TrimSpace(row[1])
		if gameURL == "" {
			continue
		}
		if currentVer == "" {
			currentVer = "—"
		}

		info, err := parser.FetchGameInfo(ctx, gameURL)
		if err != nil {
			_ = c.Send(fmt.Sprintf("Строка %d (%s): не удалось загрузить страницу: %v.", i+1, gameURL, err))
			continue
		}
		if info == nil || info.Name == "" {
			_ = c.Send(fmt.Sprintf("Строка %d (%s): не удалось получить название со страницы.", i+1, gameURL))
			continue
		}

		var latestVer *string
		if info.LatestVersion != "" {
			latestVer = &info.LatestVersion
		}
		_, err = b.gamesQuery.Create(ctx, model.Game{
			Name:           info.Name,
			WebsiteURL:     gameURL,
			CurrentVersion: currentVer,
			LatestVersion:  latestVer,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		})
		if err != nil {
			_ = c.Send(fmt.Sprintf("Строка %d (%q): %v", i+1, info.Name, err))
			continue
		}
		_ = c.Send(fmt.Sprintf("Игра %q добавлена.", info.Name))
		added++
	}

	return c.Send(fmt.Sprintf("Добавлено игр: %d.", added))
}

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

func (b *Bot) handleUpdated(c telebot.Context) error {
	payload := strings.TrimSpace(c.Message().Payload)
	if payload == "" {
		return c.Send("Использование: /updated <название1>, <название2>, ... — игры, которые вы уже обновили на устройстве.")
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

func splitNames(payload string) []string {
	var out []string
	for _, s := range strings.Split(payload, ",") {
		name := strings.TrimSpace(s)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func (b *Bot) handleRemove(c telebot.Context) error {
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
