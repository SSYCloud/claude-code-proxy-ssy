APP_NAME := claudeproxy
VERSION := 1.0.0
BUILD_DIR := dist
MAIN_FILE := main.go

# Get build info
COMMIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT_HASH) -X main.buildTime=$(BUILD_TIME) -s -w

# Default target
.PHONY: all
all: clean build

# Clean build directory
.PHONY: clean
clean:
	@echo "🧹 清理构建目录..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# Build for current platform
.PHONY: build
build:
	@echo "🔨 构建当前平台..."
	@go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_FILE)
	@echo "✅ 构建完成: $(BUILD_DIR)/$(APP_NAME)"

# Build for all platforms
.PHONY: build-all
build-all: clean
	@echo "🔨 构建所有平台..."
	@./build.sh

# Build for Linux
.PHONY: build-linux
build-linux:
	@echo "🔨 构建 Linux..."
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(MAIN_FILE)
	@GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(MAIN_FILE)

# Build for macOS
.PHONY: build-darwin
build-darwin:
	@echo "🔨 构建 macOS..."
	@GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(MAIN_FILE)
	@GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(MAIN_FILE)

# Build for Windows
.PHONY: build-windows
build-windows:
	@echo "🔨 构建 Windows..."
	@GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(MAIN_FILE)
	@GOOS=windows GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe $(MAIN_FILE)

# Install locally
.PHONY: install
install: build
	@echo "📦 安装到本地..."
	@sudo mv $(BUILD_DIR)/$(APP_NAME) /usr/local/bin/
	@echo "✅ 安装完成"

# Run development version
.PHONY: dev
dev:
	@echo "🚀 运行开发版本..."
	@go run $(MAIN_FILE)

# Run tests
.PHONY: test
test:
	@echo "🧪 运行测试..."
	@go test -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "📝 格式化代码..."
	@go fmt ./...

# Run linter
.PHONY: lint
lint:
	@echo "🔍 运行代码检查..."
	@golangci-lint run

# Show help
.PHONY: help
help:
	@echo "可用的命令:"
	@echo "  make build        - 构建当前平台"
	@echo "  make build-all    - 构建所有平台"
	@echo "  make build-linux  - 构建 Linux 平台"
	@echo "  make build-darwin - 构建 macOS 平台"
	@echo "  make build-windows- 构建 Windows 平台"
	@echo "  make install      - 安装到本地"
	@echo "  make clean        - 清理构建目录"
	@echo "  make dev          - 运行开发版本"
	@echo "  make test         - 运行测试"
	@echo "  make fmt          - 格式化代码"
	@echo "  make lint         - 运行代码检查"
