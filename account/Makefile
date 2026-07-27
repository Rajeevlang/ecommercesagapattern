DB_URL=postgres://root:secretpassword@localhost:5433/account_db?sslmode=disable

postgres:
	docker compose up -d

postgres-down:
	docker compose down

migrateup:
	migrate -path migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path migrations -database "$(DB_URL)" -verbose down

.PHONY: postgres postgres-down migrateup migratedown
