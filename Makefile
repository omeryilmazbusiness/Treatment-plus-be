.PHONY: tidy build test vet lint arch run worker migrate-up migrate-down docker-up docker-down up down sqlc vuln

BIN_DIR := .bin

tidy:
	go mod tidy

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/api ./cmd/api
	go build -o $(BIN_DIR)/worker ./cmd/worker
	go build -o $(BIN_DIR)/migrate ./cmd/migrate

vet:
	go vet ./...

arch:
	go test ./internal/arch ./internal/shared/... ./internal/platform/httpx

test:
	go test ./...

lint: vet arch

sqlc:
	sqlc generate

vuln:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

run: build
	./$(BIN_DIR)/api

worker: build
	./$(BIN_DIR)/worker

migrate-up: build
	./$(BIN_DIR)/migrate -command=up

migrate-down: build
	./$(BIN_DIR)/migrate -command=down

docker-up:
	docker compose up -d --wait

docker-down:
	docker compose down

# Laptop: infra → migrate → api. Phase 0 done when /healthz and /readyz return 200.
up: docker-up
	$(MAKE) migrate-up
	$(MAKE) run

down: docker-down
