.PHONY: build run test test-coverage clean fmt lint help

# 变量
BINARY_NAME := symphony
MAIN_PATH := ./cmd/symphony
GO := go
GOFLAGS := -v

# 默认目标
.DEFAULT_GOAL := help

## build: 构建二进制文件
build:
	$(GO) build $(GOFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)

## run: 运行服务
run: build
	./$(BINARY_NAME)

## run-dev: 使用开发配置运行服务
run-dev: build
	./$(BINARY_NAME) -workflow ./WORKFLOW_example.md -port 8080

## test: 运行所有测试
test:
	$(GO) test $(GOFLAGS) ./...

## test-coverage: 运行测试并生成覆盖率报告
test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "覆盖率报告已生成: coverage.html"

## test-coverage-term: 在终端显示测试覆盖率
test-coverage-term:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

## clean: 清理构建产物
clean:
	rm -f $(BINARY_NAME)
	rm -f coverage.out coverage.html
	rm -f *.test

## fmt: 格式化代码
fmt:
	$(GO) fmt ./...

## lint: 运行代码检查
lint:
	@which golangci-lint > /dev/null || (echo "请先安装 golangci-lint" && exit 1)
	golangci-lint run ./...

## vet: 运行 go vet
vet:
	$(GO) vet ./...

## deps: 下载依赖
deps:
	$(GO) mod download
	$(GO) mod tidy

## install: 安装到 $GOPATH/bin
install:
	$(GO) install $(MAIN_PATH)

## help: 显示帮助信息
help:
	@echo "Symphony Makefile 帮助"
	@echo ""
	@echo "用法: make [target]"
	@echo ""
	@echo "目标:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'