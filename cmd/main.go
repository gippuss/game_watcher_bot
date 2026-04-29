package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gippuss/game_watcher_bot/internal/bot"
	"github.com/gippuss/game_watcher_bot/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

const (
	configKeyBotToken = "TELEGRAM_BOT_TOKEN"
	configKeyDbUrl    = "DATABASE_URL"
	configKeyProxy    = "HTTPS_PROXY"
)

func main() {
	ctx := context.Background()

	token := os.Getenv(configKeyBotToken)
	if token == "" {
		log.Fatal("failed to get telegram bot token")
	}
	connStr := os.Getenv(configKeyDbUrl)
	if connStr == "" {
		log.Fatal("failed to get database url")
	}
	proxyStr := os.Getenv(configKeyProxy)
	proxyURL, err := url.Parse(proxyStr)
	if err != nil {
		log.Fatal(err)
	}

	if err := runMigrations(connStr); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
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

	httpProxyClient := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		},
	}

	tgBot, err := bot.New(token, httpProxyClient, gamesQuery)
	if err != nil {
		log.Fatalf("failed to create bot: %v", err)
	}

	tgBot.Start()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	tgBot.Stop()
}

func runMigrations(connStr string) error {
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	return goose.Up(db, "migrations")
}
