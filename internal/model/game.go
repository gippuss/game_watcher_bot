package model

import "time"

const (
	TableNameGames = "games"

	TableGameColumnID             = "id"
	TableGameColumnName           = "name"
	TableGameColumnWebsiteURL     = "website_url"
	TableGameColumnCurrentVersion = "current_version"
	TableGameColumnLatestVersion  = "latest_version"
	TableGameColumnCreatedAt      = "created_at"
	TableGameColumnUpdatedAt      = "updated_at"
)

type Game struct {
	ID             int64     `db:"id"`
	Name           string    `db:"name" insert:"name"`
	WebsiteURL     string    `db:"website_url" insert:"website_url"`
	CurrentVersion string    `db:"current_version" insert:"current_version"`
	LatestVersion  *string   `db:"latest_version" insert:"latest_version"`
	CreatedAt      time.Time `db:"created_at" insert:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" insert:"updated_at"`
}

type GameFilter struct {
	ID         *int64   `filter:"id"`
	Name       *string  `filter:"name"`
	Names      []string `filter:"name"`
	WebsiteURL *string  `filter:"website_url"`
}

func (g Game) HasUpdate() bool {
	if g.LatestVersion == nil || *g.LatestVersion == "" {
		return false
	}
	return *g.LatestVersion != g.CurrentVersion
}
