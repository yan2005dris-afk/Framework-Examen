# apps/backend/Makefile
#
# Tareas habituales del backend Go. La referencia completa está en `make help`.

DATABASE_URL ?= postgres://postgres@localhost:5432/biess?sslmode=disable
MIGRATE_RUN := DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate

.PHONY: help run build test vet fmt tidy sqlc-generate db-schema db-migrate db-status db-down db-reset docker-up docker-down

help:
	@echo "run             Arranca la API en modo desarrollo"
	@echo "build           Compila el binario de la API en bin/api"
	@echo "test            Ejecuta los tests"
	@echo "vet             Ejecuta go vet"
	@echo "fmt             Formatea el código con gofmt"
	@echo "tidy            Ordena el go.mod"
	@echo "sqlc-generate   Regenera el código de sqlc"
	@echo "db-schema       Regenera sql/queries/schema.sql desde las migraciones"
	@echo "db-migrate      Aplica las migraciones pendientes"
	@echo "db-status       Muestra el estado de las migraciones"
	@echo "db-down         Revierte la última migración aplicada"
	@echo "db-reset        Recrea el esquema (DESTRUCTIVO)"
	@echo "docker-up       Levanta PostgreSQL en Docker"
	@echo "docker-down     Apaga los contenedores de Docker"

run:
	go run ./cmd/api

build:
	go build -o bin/api ./cmd/api

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -l -w .

tidy:
	go mod tidy

sqlc-generate:
	sqlc generate

db-schema:
	@printf '%s\n' '-- Esquema que consume sqlc. Se regenera desde las migraciones con `make db-schema`.' > sql/queries/schema.sql
	@for file in sql/migrations/*.sql; do \
		awk '/-- \+goose Up/{up=1;next} /-- \+goose Down/{up=0} up' "$$file"; \
		echo; \
	done >> sql/queries/schema.sql
	@echo "Esquema regenerado en sql/queries/schema.sql"

db-migrate:
	$(MIGRATE_RUN)

db-status:
	$(MIGRATE_RUN) -status

db-down:
	$(MIGRATE_RUN) -down

db-reset:
	@psql "$(DATABASE_URL)" -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
	$(MIGRATE_RUN)

docker-up:
	docker compose -f ../../compose.yaml up -d

docker-down:
	docker compose -f ../../compose.yaml down
