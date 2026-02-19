package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/bot"
	"github.com/gippuss/game_watcher_bot/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	token := os.Getenv("TELEGRAM_BOT_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN не задан")
	}
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/game_watcher?sslmode=disable"
	}

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatalf("parse db config: %v", err)
	}
	poolConfig.MaxConnLifetime = 30 * time.Minute
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		log.Fatalf("error to db connect: %v", err)
	}
	defer pool.Close()

	gamesQuery, err := repository.NewGamesQuery(pool)
	if err != nil {
		log.Fatal("failed to create games query")
	}

	tgBot, err := bot.New(token, gamesQuery)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	slog.Info("bot started")
	tgBot.Start()
}
