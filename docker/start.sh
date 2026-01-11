#!/bin/bash

# Gallery 一键启动脚本
# 使用方法:
#   ./start.sh              # 启动基础服务 (数据库 + 后端 + 前端)
#   ./start.sh --with-ai    # 启动所有服务 (包含 GPU AI 模型)
#   ./start.sh --with-ai-cpu # 启动所有服务 (包含 CPU AI 模型)
#   ./start.sh --stop       # 停止所有服务
#   ./start.sh --logs       # 查看日志

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 检查 .env 文件
if [ ! -f ".env" ]; then
    echo "📋 未找到 .env 文件，从 .env.example 创建..."
    cp .env.example .env
    echo "✅ 已创建 .env 文件，请根据需要修改配置"
fi

# 解析命令行参数
case "$1" in
    --with-ai)
        echo "🚀 启动所有服务 (包含 GPU AI 模型)..."
        docker compose --profile with-ai up -d --build
        ;;
    --with-ai-cpu)
        echo "🚀 启动所有服务 (包含 CPU AI 模型)..."
        docker compose --profile with-ai-cpu up -d --build
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
        echo "  ./start.sh              启动基础服务 (数据库 + 后端 + 前端)"
        echo "  ./start.sh --with-ai    启动所有服务 (包含 GPU AI 模型)"
        echo "  ./start.sh --with-ai-cpu 启动所有服务 (包含 CPU AI 模型)"
        echo "  ./start.sh --stop       停止所有服务"
        echo "  ./start.sh --logs       查看日志"
        echo "  ./start.sh --help       显示帮助"
        exit 0
        ;;
    *)
        echo "🚀 启动基础服务 (数据库 + 后端 + 前端)..."
        docker compose up -d --build
        ;;
esac

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
if [[ "$1" == "--with-ai" || "$1" == "--with-ai-cpu" ]]; then
    echo "   AI gRPC: localhost:${AI_GRPC_PORT:-50051}"
fi
echo ""
echo "📝 查看日志: ./start.sh --logs"
echo "🛑 停止服务: ./start.sh --stop"
