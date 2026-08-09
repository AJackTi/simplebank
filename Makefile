POSTGRES_USER ?= simplebank
POSTGRES_PASSWORD ?= simplebank_dev
POSTGRES_DB ?= simple_bank
POSTGRES_PORT ?= 5432
REDIS_PORT ?= 6379
DB_URL ?= postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:$(POSTGRES_PORT)/$(POSTGRES_DB)?sslmode=disable

.DEFAULT_GOAL := help

.PHONY: help up down logs migrateup migratedown migrateup1 migratedown1 db_docs db_schema sqlc test test-race vet lint security check ci server mock proto evans

help:
	@printf '%s\n' \
		'up              Start PostgreSQL, Redis, and the application' \
		'down            Stop the local Compose stack and remove volumes' \
		'test            Run the complete test suite' \
		'test-race       Run tests with the race detector' \
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

check: vet lint test security

ci: check test-race

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
