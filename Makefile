.DEFAULT_GOAL := up

up:
	docker compose up --build

down:
	docker compose down

logs:
	docker compose logs -f --tail=100

ps:
	docker compose ps

build:
	docker compose build

generate:
	docker compose exec frontend npx openapi-typescript http://backend:8888/openapi.json -o src/api/schema.ts

fmt:
	docker compose run --rm --no-deps backend gofmt -w $$(find . -name '*.go')

test:
	docker compose run --rm --no-deps backend go test ./...

psql:
	docker compose exec db psql -U app -d app

reset:
	docker compose down -v
