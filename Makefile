.PHONY: build install test test-unit test-integration test-all test-coverage docker-up docker-down clean lint help

# Variáveis
BINARY_NAME=gohop
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# ═══════════════════════════════════════════════════════════════════════════════
# BUILD
# ═══════════════════════════════════════════════════════════════════════════════

## build: Compilar o binário
build:
	@echo "🔨 Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/gohop
	@echo "✅ Build complete: ./$(BINARY_NAME)"

## install: Instalar globalmente
install: build
	@echo "📦 Installing $(BINARY_NAME)..."
	go install ./cmd/gohop
	@echo "✅ Installed! Run 'gohop' to start"

## release: Build para múltiplas plataformas
release:
	@echo "🚀 Building releases..."
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 ./cmd/gohop
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 ./cmd/gohop
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 ./cmd/gohop
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 ./cmd/gohop
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe ./cmd/gohop
	@echo "✅ Releases built in ./dist/"

# ═══════════════════════════════════════════════════════════════════════════════
# TESTES
# ═══════════════════════════════════════════════════════════════════════════════

## test: Executar todos os testes unitários
test:
	@echo "🧪 Running tests..."
	go test -v ./...

## test-unit: Executar apenas testes unitários (sem integração)
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -short ./...

## test-integration: Executar testes de integração (requer Docker)
test-integration: docker-up
	@echo "🧪 Running integration tests..."
	@sleep 5
	go test -v -tags=integration ./...
	@$(MAKE) docker-down

## test-all: Executar TODOS os testes (unitários + integração)
test-all: docker-up
	@echo "🧪 Running all tests..."
	@sleep 5
	go test -v -tags=integration ./...
	@$(MAKE) docker-down

## test-coverage: Executar testes com cobertura
test-coverage:
	@echo "📊 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# ═══════════════════════════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════════════════════════

## docker-up: Subir RabbitMQ para testes
docker-up:
	@echo "🐰 Starting RabbitMQ..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "⏳ Waiting for RabbitMQ..."
	@timeout=60; \
	while [ $$timeout -gt 0 ]; do \
		if docker-compose -f docker-compose.test.yml exec -T rabbitmq rabbitmq-diagnostics ping >/dev/null 2>&1; then \
			echo "✅ RabbitMQ is ready!"; \
			break; \
		fi; \
		sleep 2; \
		timeout=$$((timeout-2)); \
	done

## docker-down: Parar RabbitMQ
docker-down:
	@echo "🛑 Stopping RabbitMQ..."
	docker-compose -f docker-compose.test.yml down -v

# ═══════════════════════════════════════════════════════════════════════════════
# DESENVOLVIMENTO
# ═══════════════════════════════════════════════════════════════════════════════

## run: Executar a CLI
run: build
	./$(BINARY_NAME)

## lint: Executar linter
lint:
	@echo "🔍 Running linter..."
	golangci-lint run ./...

## fmt: Formatar código
fmt:
	@echo "🎨 Formatting code..."
	go fmt ./...
	goimports -w .

## tidy: Limpar dependências
tidy:
	@echo "🧹 Tidying dependencies..."
	go mod tidy

## clean: Limpar arquivos gerados
clean:
	@echo "🗑️  Cleaning..."
	rm -f $(BINARY_NAME) coverage.out coverage.html
	rm -rf dist/
	go clean

# ═══════════════════════════════════════════════════════════════════════════════
# AJUDA
# ═══════════════════════════════════════════════════════════════════════════════

## help: Mostrar ajuda
help:
	@echo ""
	@echo "🐰 GoHop - RabbitMQ CLI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build          Build the binary"
	@echo "  install        Install globally"
	@echo "  release        Build for all platforms"
	@echo ""
	@echo "Test:"
	@echo "  test           Run all unit tests"
	@echo "  test-unit      Run unit tests only"
	@echo "  test-integration  Run integration tests (requires Docker)"
	@echo "  test-all       Run all tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up      Start RabbitMQ for testing"
	@echo "  docker-down    Stop RabbitMQ"
	@echo ""
	@echo "Development:"
	@echo "  run            Build and run"
	@echo "  lint           Run linter"
	@echo "  fmt            Format code"
	@echo "  tidy           Tidy dependencies"
	@echo "  clean          Clean generated files"
	@echo ""

.DEFAULT_GOAL := help
