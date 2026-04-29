package bot

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/model"
	"github.com/gippuss/game_watcher_bot/internal/parser"
	"gopkg.in/telebot.v3"
)

type uploadTask struct {
	rowNum     int
	gameURL    string
	currentVer string
}

func (b *bot) handleUpload(c telebot.Context) error {
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
	var sendMu sync.Mutex
	sem := make(chan struct{}, b.concurrency)
	var wg sync.WaitGroup
	var added atomic.Int64

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

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			b.processUploadRow(ctx, c, uploadTask{rowNum: i + 1, gameURL: gameURL, currentVer: currentVer}, &added, &sendMu)
		}()
	}

	wg.Wait()
	return c.Send(fmt.Sprintf("Добавлено игр: %d.", added.Load()))
}

func (b *bot) processUploadRow(ctx context.Context, c telebot.Context, task uploadTask, added *atomic.Int64, sendMu *sync.Mutex) {
	info, err := parser.FetchGameInfo(ctx, task.gameURL)
	if err != nil {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("Строка %d (%s): не удалось загрузить страницу: %v.", task.rowNum, task.gameURL, err))
		sendMu.Unlock()
		return
	}
	if info == nil || info.Name == "" {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("Строка %d (%s): не удалось получить название со страницы.", task.rowNum, task.gameURL))
		sendMu.Unlock()
		return
	}

	var latestVer *string
	if info.LatestVersion != "" {
		latestVer = &info.LatestVersion
	}
	_, err = b.gamesQuery.Create(ctx, model.Game{
		Name:           info.Name,
		WebsiteURL:     task.gameURL,
		CurrentVersion: task.currentVer,
		LatestVersion:  latestVer,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
	if err != nil {
		sendMu.Lock()
		_ = c.Send(fmt.Sprintf("Строка %d (%q): %v", task.rowNum, info.Name, err))
		sendMu.Unlock()
		return
	}
	sendMu.Lock()
	_ = c.Send(fmt.Sprintf("Игра %q добавлена.", info.Name))
	sendMu.Unlock()
	added.Add(1)
}
