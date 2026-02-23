.PHONY: docker-up docker-down logs-app docker-build-app db

docker-up:
	docker compose up -d

docker-down:
	docker compose down

logs-app:
	docker compose logs -f app

docker-build-app:
	docker compose up -d --build app

db:
	docker compose exec db psql -U game_watcher -d game_watcher
