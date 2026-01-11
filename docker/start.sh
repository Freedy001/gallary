#!/bin/bash

# Gallery 一键启动脚本
# 使用方法:
#   ./start.sh                    # 启动基础服务 (数据库 + 后端 + 前端)
#   ./start.sh --with-ai          # 启动所有服务 (包含 GPU AI 模型)
#   ./start.sh --with-ai-cpu      # 启动所有服务 (包含 CPU AI 模型)
#   ./start.sh --prebuilt <path>  # 使用预编译的 server 二进制文件
#   ./start.sh --stop             # 停止所有服务
#   ./start.sh --logs             # 查看日志

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 默认值
USE_PREBUILT=false
PREBUILT_PATH=""
COMPOSE_FILE="docker-compose.yml"
PROFILES=""

# 检查 .env 文件
if [ ! -f ".env" ]; then
    echo "📋 未找到 .env 文件，从 .env.example 创建..."
    cp .env.example .env
    echo "✅ 已创建 .env 文件，请根据需要修改配置"
fi

# 加载环境变量
source .env 2>/dev/null || true

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case "$1" in
        --prebuilt)
            USE_PREBUILT=true
            PREBUILT_PATH="$2"
            shift 2
            ;;
        --with-ai)
            PROFILES="--profile with-ai"
            shift
            ;;
        --with-ai-cpu)
            PROFILES="--profile with-ai-cpu"
            shift
            ;;
        --stop)
            echo "🛑 停止所有服务..."
            docker compose --profile with-ai --profile with-ai-cpu down
            echo "✅ 服务已停止"
            exit 0
            ;;
        --logs)
            docker compose logs -f
            exit 0
            ;;
        --help)
            echo "Gallery Docker 启动脚本"
            echo ""
            echo "使用方法:"
            echo "  ./start.sh                      启动基础服务 (数据库 + 后端 + 前端)"
            echo "  ./start.sh --with-ai            启动所有服务 (包含 GPU AI 模型)"
            echo "  ./start.sh --with-ai-cpu        启动所有服务 (包含 CPU AI 模型)"
            echo "  ./start.sh --prebuilt <path>    使用预编译的 server 二进制文件"
            echo "  ./start.sh --stop               停止所有服务"
            echo "  ./start.sh --logs               查看日志"
            echo "  ./start.sh --help               显示帮助"
            echo ""
            echo "示例:"
            echo "  # 本地交叉编译后端，然后使用预编译文件启动"
            echo "  cd ../server && GOOS=linux GOARCH=amd64 go build -o ../docker/bin/server ./main.go"
            echo "  ./start.sh --prebuilt ./bin/server"
            echo ""
            echo "  # 同时使用预编译文件和 CPU AI 模型"
            echo "  ./start.sh --prebuilt ./bin/server --with-ai-cpu"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 --help 查看帮助"
            exit 1
            ;;
    esac
done

# 处理预编译二进制文件
if [ "$USE_PREBUILT" = true ]; then
    if [ -z "$PREBUILT_PATH" ]; then
        echo "❌ 错误: --prebuilt 需要指定二进制文件路径"
        exit 1
    fi

    if [ ! -f "$PREBUILT_PATH" ]; then
        echo "❌ 错误: 找不到二进制文件: $PREBUILT_PATH"
        exit 1
    fi

    echo "📦 使用预编译的 server 二进制文件: $PREBUILT_PATH"

    # 创建 bin 目录并复制二进制文件
    mkdir -p ./bin
    cp "$PREBUILT_PATH" ./bin/server
    chmod +x ./bin/server

    # 使用预编译版本的 compose 文件
    COMPOSE_FILE="docker-compose.yml -f docker-compose.prebuilt.yml"

    echo "✅ 已复制二进制文件到 ./bin/server"
fi

# 启动服务
if [ -n "$PROFILES" ]; then
    echo "🚀 启动服务 (包含 AI 模型)..."
else
    echo "🚀 启动基础服务 (数据库 + 后端 + 前端)..."
fi

docker compose -f $COMPOSE_FILE $PROFILES up -d --build

echo ""
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
echo ""
echo "📊 服务状态:"
docker compose ps

echo ""
echo "✅ 启动完成!"
echo ""
echo "🌐 访问地址:"
echo "   前端:   http://localhost:${FRONTEND_PORT:-80}"
echo "   后端:   http://localhost:${SERVER_PORT:-9099}"
echo "   数据库: localhost:${DB_PORT:-5432}"
if [[ "$PROFILES" == *"with-ai"* ]]; then
    echo "   AI gRPC: localhost:${AI_GRPC_PORT:-50051}"
fi
echo ""
echo "📝 查看日志: ./start.sh --logs"
echo "🛑 停止服务: ./start.sh --stop"
