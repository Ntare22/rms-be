.PHONY: all build run run-worker test itest swagger clean docker-up docker-down

all: build test

BIN_DIR := bin

build:
	@echo "Building API and worker..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/rms-api ./cmd/api
	@go build -o $(BIN_DIR)/rms-worker ./cmd/worker

run:
	@go run ./cmd/api

run-worker:
	@go run ./cmd/worker

test:
	@go test ./... -short -count=1

itest:
	@go test ./internal/database -count=1

swagger:
	@swag init -g main.go -d ./cmd/api,./internal/app,./internal/modules/auth,./internal/modules/organizations,./internal/modules/users,./internal/modules/buildings,./internal/modules/units,./internal/modules/tenants,./internal/modules/leases,./internal/modules/payments,./internal/modules/sms -o ./docs --parseDependency --parseInternal

clean:
	@rm -rf $(BIN_DIR) main

docker-up:
	@if docker compose up -d 2>/dev/null; then \
		: ; \
	else \
		docker-compose up -d; \
	fi

docker-down:
	@if docker compose down 2>/dev/null; then \
		: ; \
	else \
		docker-compose down; \
	fi

watch:
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "Install air: go install github.com/air-verse/air@latest"; \
		exit 1; \
	fi
