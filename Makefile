.PHONY: all test bench clean lint install-lint

# 统一设置 CGO_ENABLED=0 (对跨平台构建通常有效，但注意 -race 需要 CGO)
export CGO_ENABLED=0

# -----------------------------------------------------------------------------
# 操作系统检测与命令适配
# -----------------------------------------------------------------------------
ifeq ($(OS),Windows_NT)
    # Windows 环境 (CMD/PowerShell)
    # 注意：Windows 下使用 del 删除文件，/Q 不确认，/F 强制
    RM = del /Q /F
    # Windows 下检测命令较复杂，直接运行 go install 确保安装
    ENSURE_LINT = go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    # Windows: 不使用 -race，避免 cgo 报错
    TEST_CMD = go test -v -race ./...
else
    # Linux / macOS (Unix-like)
    RM = rm -f
    # 检测是否安装，未安装则执行安装
    ENSURE_LINT = which golangci-lint >/dev/null 2>&1 || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    # Linux/Mac: 启用 -race (需要临时开启 CGO)
    TEST_CMD = CGO_ENABLED=1 go test -v -race ./...
endif

# -----------------------------------------------------------------------------
# 目标定义
# -----------------------------------------------------------------------------

# 默认目标
all: test bench

# 运行所有单元测试 (使用适配后的 TEST_CMD)
test:
	$(TEST_CMD)

# 运行基准测试
bench:
	go test -bench="." -benchmem ./...

# 辅助目标：确保 golangci-lint 已安装
install-lint:
	@echo "Checking/Installing golangci-lint..."
	@$(ENSURE_LINT)

# 代码格式化与检查
# 依赖 install-lint 目标，运行前会自动检查工具是否存在
lint: install-lint
	go fmt ./...
	go vet ./...
	golangci-lint run ./...

# 清理临时文件
clean:
	go clean
	# 前面的 '-' 表示忽略错误 (例如文件不存在时不报错)
	-$(RM) coverage.out coverage.txt
