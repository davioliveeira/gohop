.PHONY: build install test test-unit test-integration test-all test-coverage docker-up docker-down clean lint help

# Variáveis
BINARY_NAME=gohop
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Pacotes para cobertura (exclui UI que é difícil de testar unitariamente)
COVERAGE_PACKAGES=./internal/config/... ./internal/rabbitmq/... ./internal/retry/... ./cmd/commands/...

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
	@go install ./cmd/gohop
	@GOBIN_PATH=$$(go env GOPATH)/bin; \
	if ! echo "$$PATH" | grep -q "$$GOBIN_PATH"; then \
		echo "🔧 Configuring PATH..."; \
		SHELL_RC=""; \
		if [ -f "$$HOME/.zshrc" ]; then \
			SHELL_RC="$$HOME/.zshrc"; \
		elif [ -f "$$HOME/.bashrc" ]; then \
			SHELL_RC="$$HOME/.bashrc"; \
		elif [ -f "$$HOME/.bash_profile" ]; then \
			SHELL_RC="$$HOME/.bash_profile"; \
		fi; \
		if [ -n "$$SHELL_RC" ]; then \
			if ! grep -q 'go/bin' "$$SHELL_RC" 2>/dev/null; then \
				echo "" >> "$$SHELL_RC"; \
				echo "# GoHop - Go binaries path" >> "$$SHELL_RC"; \
				echo 'export PATH="$$PATH:$$HOME/go/bin"' >> "$$SHELL_RC"; \
				echo "✅ Added Go bin to PATH in $$SHELL_RC"; \
				echo "⚠️  Run 'source $$SHELL_RC' or open a new terminal"; \
			fi; \
		fi; \
	fi
	@echo ""
	@echo "✅ Installed! Run 'gohop' to start"
	@echo "   Location: $$(go env GOPATH)/bin/$(BINARY_NAME)"

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
	@echo "🧪 Running unit tests..."
	go test -v -short ./...

## test-unit: Executar apenas testes unitários (sem integração)
test-unit:
	@echo "🧪 Running unit tests..."
	go test -v -short ./...

## test-integration: Executar testes de integração (requer Docker)
test-integration: docker-up
	@echo "🧪 Running integration tests..."
	@sleep 3
	go test -v -tags=integration ./internal/rabbitmq/... ./internal/retry/...
	@$(MAKE) docker-down

## test-all: Executar TODOS os testes (unitários + integração)
test-all: docker-up
	@echo "🧪 Running all tests..."
	@sleep 3
	go test -v -tags=integration ./...
	@$(MAKE) docker-down

## test-coverage: Executar testes com cobertura (exclui UI)
test-coverage:
	@echo "📊 Running tests with coverage (excluding UI)..."
	go test -v -coverprofile=coverage.out -covermode=atomic $(COVERAGE_PACKAGES)
	@echo ""
	@echo "📈 Coverage Summary:"
	@go tool cover -func=coverage.out | grep total
	@echo ""
	@echo "📄 Generating HTML report..."
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

## test-coverage-integration: Cobertura com testes de integração
test-coverage-integration: docker-up
	@echo "📊 Running coverage with integration tests..."
	@sleep 3
	go test -v -tags=integration -coverprofile=coverage.out -covermode=atomic $(COVERAGE_PACKAGES)
	@$(MAKE) docker-down
	@echo ""
	@echo "📈 Coverage Summary:"
	@go tool cover -func=coverage.out | grep total
	@echo ""
	go tool cover -html=coverage.out -o coverage.html
	@echo "✅ Coverage report: coverage.html"

# ═══════════════════════════════════════════════════════════════════════════════
# DOCKER
# ═══════════════════════════════════════════════════════════════════════════════

## docker-up: Subir RabbitMQ para testes
docker-up:
	@echo "🐰 Starting RabbitMQ..."
	@docker-compose -f docker-compose.test.yml up -d
	@echo "⏳ Waiting for RabbitMQ to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15; do \
		if docker-compose -f docker-compose.test.yml exec -T rabbitmq rabbitmq-diagnostics ping >/dev/null 2>&1; then \
			echo "✅ RabbitMQ is ready!"; \
			exit 0; \
		fi; \
		echo "   Waiting... ($$i/15)"; \
		sleep 2; \
	done; \
	echo "⚠️  RabbitMQ may not be fully ready, but continuing..."

## docker-down: Parar RabbitMQ
docker-down:
	@echo "🛑 Stopping RabbitMQ..."
	@docker-compose -f docker-compose.test.yml down -v 2>/dev/null || true

## docker-logs: Ver logs do RabbitMQ
docker-logs:
	docker-compose -f docker-compose.test.yml logs -f rabbitmq

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
	@echo "  build              Build the binary"
	@echo "  install            Install globally"
	@echo "  release            Build for all platforms"
	@echo ""
	@echo "Test:"
	@echo "  test               Run unit tests"
	@echo "  test-unit          Run unit tests only"
	@echo "  test-integration   Run integration tests (Docker)"
	@echo "  test-all           Run all tests (unit + integration)"
	@echo "  test-coverage      Coverage report (excludes UI)"
	@echo "  test-coverage-integration  Coverage with integration tests"
	@echo ""
	@echo "Docker:"
	@echo "  docker-up          Start RabbitMQ for testing"
	@echo "  docker-down        Stop RabbitMQ"
	@echo "  docker-logs        View RabbitMQ logs"
	@echo ""
	@echo "Development:"
	@echo "  run                Build and run"
	@echo "  lint               Run linter"
	@echo "  fmt                Format code"
	@echo "  tidy               Tidy dependencies"
	@echo "  clean              Clean generated files"
	@echo ""

.DEFAULT_GOAL := help
