DB_URL=postgres://root:secretpassword@localhost:5432/catalog_db?sslmode=disable

build:
	go build -o catalog-service cmd/server/main.go

migrateup:
	migrate -path migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path migrations -database "$(DB_URL)" -verbose down

.PHONY: build migrateup migratedown
