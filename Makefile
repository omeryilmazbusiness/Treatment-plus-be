.PHONY: tidy build test vet lint arch run worker migrate-up migrate-down docker-up docker-down up down sqlc vuln mouth

BIN_DIR := .bin
MOUTH := internal/viz/player/static/mouth.glb
MOUTH_URL := https://github.com/omeryilmazbusiness/Treatment-plus-be/releases/download/mouth-glb/mouth.glb

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

# 143MB mouth.glb is gitignored (GitHub 100MB git limit). Collaborators get it from the public Release.
mouth:
	@if [ -f "$(MOUTH)" ]; then echo "mouth.glb already present"; \
	else curl -L --fail --retry 3 -o "$(MOUTH)" "$(MOUTH_URL)"; fi

# Laptop: infra → model → migrate → api. Phase 0 done when /healthz and /readyz return 200.
up: docker-up
	$(MAKE) mouth
	$(MAKE) migrate-up
	$(MAKE) run

down: docker-down
