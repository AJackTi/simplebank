POSTGRES_USER ?= simplebank
POSTGRES_PASSWORD ?= simplebank_dev
POSTGRES_DB ?= simple_bank
POSTGRES_PORT ?= 5432
REDIS_PORT ?= 6379
DB_URL ?= postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable
GITLEAKS_IMAGE ?= zricethezav/gitleaks:v8.30.1

.DEFAULT_GOAL := help

.PHONY: help up down logs migrateup migratedown migrateup1 migratedown1 db_docs db_schema sqlc test test-race vet lint security gitleaks docker-build check ci server mock proto evans

help:
	@printf '%s\n' \
		'up              Start PostgreSQL, Redis, and the application' \
		'down            Stop the local Compose stack and remove volumes' \
		'test            Run the complete test suite' \
		'test-race       Run tests with the race detector' \
		'gitleaks        Scan the repository history for secrets' \
		'docker-build    Build the release container image' \
		'check           Run vet, lint, tests, and security checks' \
		'proto           Regenerate protobuf and OpenAPI artifacts'

up:
	docker compose up -d

down:
	docker compose down -v --remove-orphans

logs:
	docker compose logs -f api

migrateup:
	migrate -path db/migration --database "$(DB_URL)" --verbose up

migratedown:
	migrate -path db/migration --database "$(DB_URL)" --verbose down

migrateup1:
	migrate -path db/migration --database "$(DB_URL)" --verbose up 1

migratedown1:
	migrate -path db/migration --database "$(DB_URL)" --verbose down 1

db_docs:
	dbdocs build doc/db.dbml

db_schema:
	dbml2sql --postgres -o doc/schema.sql doc/db.dbml

sqlc:
	sqlc generate

test:
	SIMPLEBANK_TEST_DB_SOURCE="$(DB_URL)" go test -v -cover ./...

test-race:
	SIMPLEBANK_TEST_DB_SOURCE="$(DB_URL)" go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

security:
	govulncheck -show verbose ./...

gitleaks:
	docker run --rm -v "$(CURDIR):/path" $(GITLEAKS_IMAGE) detect --source /path --no-banner --redact

docker-build:
	docker build --pull -t simplebank:local .

check: vet lint test security

ci: check test-race gitleaks docker-build

server:
	go run .

mock:
	go run go.uber.org/mock/mockgen@v0.6.0 -package mockdb -destination db/mock/store.go github.com/AJackTi/simplebank/db/sqlc Store

proto:
	protoc --proto_path=proto --go_out=pb --go_opt=paths=source_relative \
		--go-grpc_out=pb --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=pb --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=doc/swagger --openapiv2_opt=allow_merge=true,merge_file_name=simple_bank \
		proto/*.proto

evans:
	evans --host localhost --port 9090 -r repl
