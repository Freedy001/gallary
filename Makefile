.PHONY: help install run build clean test

help: ## 显示帮助信息
	@echo "可用命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## 安装依赖
	go mod download
	go mod tidy

run: ## 运行程序
	go run cmd/main.go

build: ## 编译程序
	go build -o bin/gallery cmd/main.go

clean: ## 清理编译文件
	rm -rf bin/
	rm -rf storage/
	rm -rf logs/

test: ## 运行测试
	go test -v ./...

dev: ## 开发模式运行（自动重启）
	go run cmd/main.go

docker-build: ## 构建 Docker 镜像
	docker build -t gallery-server .

docker-run: ## 运行 Docker 容器
	docker run -p 8080:8080 -v $(PWD)/config.yaml:/app/config.yaml gallery-server
