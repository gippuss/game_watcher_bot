package bot

import (
	"github.com/gippuss/game_watcher_bot/internal/repository"
	"gopkg.in/telebot.v3"
)

type Bot struct {
	tg         *telebot.Bot
	gamesQuery repository.GamesQuery
}

func New(token string, gamesQuery repository.GamesQuery) (*Bot, error) {
	tg, err := telebot.NewBot(telebot.Settings{Token: token})
	if err != nil {
		return nil, err
	}

	b := &Bot{tg: tg, gamesQuery: gamesQuery}

	tg.Handle("/start", b.handleStart)
	tg.Handle("/upload", b.handleUpload)
	tg.Handle(telebot.OnDocument, b.handleUpload)
	tg.Handle("/check", b.handleCheck)
	tg.Handle("/list", b.handleList)
	tg.Handle("/updated", b.handleUpdated)
	tg.Handle("/remove", b.handleRemove)

	return b, nil
}

func (b *Bot) Start() {
	b.tg.Start()
}
