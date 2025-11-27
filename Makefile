.PHONY: all test bench clean lint

export CGO_ENABLED=0

# 默认目标
all: test bench

# 运行所有单元测试
test:
	go test -v ./...

# 运行基准测试 (你的项目核心)
bench:
	go test -bench=. -benchmem ./...

# 代码格式化与检查
lint:
	go fmt ./...
	go vet ./...
	# 如果安装了 golangci-lint 可以取消下面这行的注释
	# golangci-lint run

# 清理临时文件
clean:
	go clean
	rm -f coverage.out
