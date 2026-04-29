package bot

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gippuss/game_watcher_bot/internal/repository"
	"gopkg.in/telebot.v3"
)

type Bot interface {
	Start()
	Stop()
}

type bot struct {
	tg         *telebot.Bot
	gamesQuery repository.GamesQuery

	concurrency int
}

func New(token string, proxyClient *http.Client, gamesQuery repository.GamesQuery) (Bot, error) {
	tg, err := telebot.NewBot(telebot.Settings{
		Token:  token,
		Client: proxyClient,
	})
	if err != nil {
		return nil, err
	}

	b := &bot{
		tg:          tg,
		gamesQuery:  gamesQuery,
		concurrency: concurrencyFromEnv(),
	}

	tg.Handle("/start", b.handleStart)
	tg.Handle("/upload", b.handleUpload)
	tg.Handle(telebot.OnDocument, b.handleUpload)
	tg.Handle("/check", b.handleCheck)
	tg.Handle("/list", b.handleList)
	tg.Handle("/updated", b.handleUpdated)
	tg.Handle("/remove", b.handleRemove)

	return b, nil
}

func (b *bot) Start() {
	slog.Info("bot started")
	slog.Info(fmt.Sprintf("level of concurrency: %d", b.concurrency))
	b.tg.Start()
}

func (b *bot) Stop() {
	slog.Info("bot stoped")
	b.tg.Stop()
}
