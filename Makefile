postgres:
	docker run --name postgres12 --network bank-network -e POSTGRES_USER=root -e POSTGRES_PASSWORD=password -p 5433:5432 -d postgres:12-alpine

createdb:
	docker exec -it postgres12 createdb --username=root --owner=root simple_bank

dropdb:
	docker exec -it postgres12 dropdb simple_bank

migrateup:
	migrate -path db/migration --database "postgresql://root:password@localhost:5433/simple_bank?sslmode=disable" --verbose up

migrateup1:
	migrate -path db/migration --database "postgresql://root:password@localhost:5433/simple_bank?sslmode=disable" --verbose up 1

migratedown:
	migrate -path db/migration --database "postgresql://root:password@localhost:5433/simple_bank?sslmode=disable" --verbose down

migratedown1:
	migrate -path db/migration --database "postgresql://root:password@localhost:5433/simple_bank?sslmode=disable" --verbose down 1

sqlc:
	sqlc generate

test:
	go test -v -cover ./...

server:
	go run main.go

mock:
	mockgen -package mockdb -destination db/mock/store.go github.com/ajackti/simplebank/db/sqlc Store

.PHONY: postgres createdb dropdb migrateup migratedown migrateup1 migratedown1 sqlc test server mock