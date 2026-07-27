DB_URL=postgres://root:secretpassword@localhost:5432/auth_db?sslmode=disable

postgres:
	docker compose up -d

postgres-down:
	docker compose down

migrateup:
	migrate -path migrations -database "$(DB_URL)" -verbose up

migratedown:
	migrate -path migrations -database "$(DB_URL)" -verbose down

sqlc:
	sqlc generate

.PHONY: postgres postgres-down migrateup migratedown sqlc
