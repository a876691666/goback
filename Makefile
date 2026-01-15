.PHONY: all build clean run test lint docker

# Go相关变量
GO=go
GOFMT=gofmt
GOTEST=$(GO) test
GOBUILD=$(GO) build

# 项目变量
PROJECT_NAME=goback
ifeq ($(OS),Windows_NT)
  VERSION=$(shell git describe --tags --always --dirty 2>NUL || echo v0.0.0)
  BUILD_TIME=$(shell powershell -NoProfile -Command "(Get-Date).ToString('yyyy-MM-ddTHH:mm:ssK')")
  ENV_PREFIX=set "CGO_ENABLED=0"&&
  MKDIR_BIN=if not exist $(OUTPUT_DIR) mkdir $(OUTPUT_DIR)
  RM_BIN=if exist $(OUTPUT_DIR) rmdir /S /Q $(OUTPUT_DIR)
  RM_COV=if exist coverage.out del /Q coverage.out & if exist coverage.html del /Q coverage.html
else
  VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.0)
  BUILD_TIME=$(shell date +%FT%T%z)
  ENV_PREFIX=CGO_ENABLED=0
  MKDIR_BIN=mkdir -p $(OUTPUT_DIR)
  RM_BIN=rm -rf $(OUTPUT_DIR)
  RM_COV=rm -f coverage.out coverage.html
endif
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# 服务列表
SERVICES=gateway user rbac menu log dict config

# 输出目录
OUTPUT_DIR=bin

# 默认目标
all: build

# 安装依赖
deps:
	$(GO) mod download
	$(GO) mod tidy

# 格式化代码
fmt:
	$(GOFMT) -s -w .

# 代码检查
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint is not installed, skipping..."; \
	fi

# 运行测试
test:
	$(GOTEST) -v -race -cover ./...

# 测试覆盖率报告
cover:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 构建所有服务
build: $(SERVICES)

$(SERVICES):
	@echo "Building $@..."
	@$(MKDIR_BIN)
	$(ENV_PREFIX) $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$@ ./services/$@/cmd

# 构建单个服务
build-%:
	@echo "Building $*..."
	@$(MKDIR_BIN)
	$(ENV_PREFIX) $(GOBUILD) $(LDFLAGS) -o $(OUTPUT_DIR)/$* ./services/$*/cmd

# 运行服务
run-%:
	$(GO) run ./services/$*/cmd

# 清理
clean:
	@$(RM_BIN)
	@$(RM_COV)

# Docker构建
docker-build:
	@for service in $(SERVICES); do \
		echo "Building Docker image for $$service..."; \
		docker build -t $(PROJECT_NAME)/$$service:$(VERSION) -f deployments/docker/Dockerfile.$$service .; \
	done

# Docker构建单个服务
docker-build-%:
	docker build -t $(PROJECT_NAME)/$*:$(VERSION) -f deployments/docker/Dockerfile.$* .

# Docker Compose启动
docker-up:
	docker-compose -f deployments/docker/docker-compose.yml up -d

# Docker Compose停止
docker-down:
	docker-compose -f deployments/docker/docker-compose.yml down

# 生成Swagger文档
swagger:
	@if command -v swag >/dev/null 2>&1; then \
		swag init -g services/gateway/cmd/main.go -o docs/swagger; \
	else \
		echo "swag is not installed. Run: go install github.com/swaggo/swag/cmd/swag@latest"; \
	fi

# 数据库迁移
migrate-up:
	@echo "Running database migrations..."
	$(GO) run scripts/migrate.go up

migrate-down:
	@echo "Rolling back database migrations..."
	$(GO) run scripts/migrate.go down

# 代码生成
generate:
	$(GO) generate ./...

# 帮助
help:
	@echo "Available targets:"
	@echo "  all          - Build all services (default)"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  test         - Run tests"
	@echo "  cover        - Generate coverage report"
	@echo "  build        - Build all services"
	@echo "  build-<svc>  - Build specific service (gateway, user, rbac, menu, log, dict)"
	@echo "  run-<svc>    - Run specific service"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-build - Build all Docker images"
	@echo "  docker-up    - Start services with Docker Compose"
	@echo "  docker-down  - Stop services with Docker Compose"
	@echo "  swagger      - Generate Swagger documentation"
	@echo "  help         - Show this help"
