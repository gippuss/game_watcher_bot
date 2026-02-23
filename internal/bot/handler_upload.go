package bot

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/gippuss/game_watcher_bot/internal/parser"
	"gopkg.in/telebot.v3"
)

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
