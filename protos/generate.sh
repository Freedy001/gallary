#!/bin/bash

# gRPC 代码生成脚本
# 用于从 .proto 文件生成 Go 和 Python 的 gRPC 代码

set -e  # 遇到错误时退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# 路径配置
PROTO_DIR="$SCRIPT_DIR"
PROTO_FILE="ai.proto"
GO_OUT_DIR="$PROJECT_ROOT/server/grpc"
PYTHON_OUT_DIR="$PROJECT_ROOT/model/app/grpc"

echo -e "${GREEN}=== gRPC 代码生成脚本 ===${NC}"
echo "项目根目录: $PROJECT_ROOT"
echo "Proto 文件: $PROTO_DIR/$PROTO_FILE"
echo "Go 输出目录: $GO_OUT_DIR"
echo "Python 输出目录: $PYTHON_OUT_DIR"
echo ""

# 检查 proto 文件是否存在
if [ ! -f "$PROTO_DIR/$PROTO_FILE" ]; then
    echo -e "${RED}错误: Proto 文件不存在: $PROTO_DIR/$PROTO_FILE${NC}"
    exit 1
fi

# 创建输出目录
mkdir -p "$GO_OUT_DIR"
mkdir -p "$PYTHON_OUT_DIR"

# ============== 检查依赖 ==============

check_command() {
    if ! command -v $1 &> /dev/null; then
        echo -e "${RED}错误: $1 未安装${NC}"
        echo -e "${YELLOW}安装提示: $2${NC}"
        return 1
    fi
    echo -e "${GREEN}✓ $1 已安装${NC}"
    return 0
}

echo -e "${YELLOW}检查依赖...${NC}"

# 检查 protoc
if ! check_command protoc "请访问 https://grpc.io/docs/protoc-installation/ 安装 protoc"; then
    exit 1
fi

# 检查 protoc-gen-go (Go gRPC 插件)
if ! check_command protoc-gen-go "运行: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"; then
    GO_PLUGIN_MISSING=1
fi

# 检查 protoc-gen-go-grpc
if ! check_command protoc-gen-go-grpc "运行: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"; then
    GO_PLUGIN_MISSING=1
fi

if [ "$GO_PLUGIN_MISSING" = "1" ]; then
    echo -e "${RED}Go gRPC 插件缺失,无法生成 Go 代码${NC}"
    exit 1
fi

# 检查 Python grpcio-tools
if ! python3 -c "import grpc_tools.protoc" 2>/dev/null; then
    echo -e "${RED}错误: Python grpcio-tools 未安装${NC}"
    echo -e "${YELLOW}安装提示: pip install grpcio-tools${NC}"
    exit 1
else
    echo -e "${GREEN}✓ Python grpcio-tools 已安装${NC}"
fi

echo ""

# ============== 生成 Go 代码 ==============

echo -e "${YELLOW}生成 Go gRPC 代码...${NC}"

# 清理旧文件
rm -f "$GO_OUT_DIR"/*.go

# 生成 Go 代码
protoc \
    --proto_path="$PROTO_DIR" \
    --go_out="$GO_OUT_DIR" \
    --go_opt=paths=source_relative \
    --go-grpc_out="$GO_OUT_DIR" \
    --go-grpc_opt=paths=source_relative \
    "$PROTO_DIR/$PROTO_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Go 代码生成成功${NC}"
    echo "  生成文件:"
    ls -lh "$GO_OUT_DIR"/*.go | awk '{print "    " $9 " (" $5 ")"}'
else
    echo -e "${RED}✗ Go 代码生成失败${NC}"
    exit 1
fi

echo ""

# ============== 生成 Python 代码 ==============

echo -e "${YELLOW}生成 Python gRPC 代码...${NC}"

# 清理旧文件和缓存
rm -f "$PYTHON_OUT_DIR"/ai_pb2*.py*
rm -rf "$PYTHON_OUT_DIR"/__pycache__

# 生成 Python 代码
python3 -m grpc_tools.protoc \
    --proto_path="$PROTO_DIR" \
    --python_out="$PYTHON_OUT_DIR" \
    --grpc_python_out="$PYTHON_OUT_DIR" \
    --pyi_out="$PYTHON_OUT_DIR" \
    "$PROTO_DIR/$PROTO_FILE"

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Python 代码生成成功${NC}"

    # 修复 Python 导入问题：将绝对导入改为相对导入
    echo -e "${YELLOW}修复 Python 导入...${NC}"

    # 修复 ai_pb2_grpc.py 中的导入
    if [ -f "$PYTHON_OUT_DIR/ai_pb2_grpc.py" ]; then
        # macOS 的 sed 需要 -i '' 参数
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' 's/^import ai_pb2 as ai__pb2$/from . import ai_pb2 as ai__pb2/g' "$PYTHON_OUT_DIR/ai_pb2_grpc.py"
        else
            sed -i 's/^import ai_pb2 as ai__pb2$/from . import ai_pb2 as ai__pb2/g' "$PYTHON_OUT_DIR/ai_pb2_grpc.py"
        fi
        echo -e "${GREEN}✓ 导入修复完成${NC}"
    fi

    echo "  生成文件:"
    ls -lh "$PYTHON_OUT_DIR"/ai_pb2* 2>/dev/null | awk '{print "    " $9 " (" $5 ")"}'
else
    echo -e "${RED}✗ Python 代码生成失败${NC}"
    exit 1
fi

echo ""

# ============== 创建 Python __init__.py ==============

if [ ! -f "$PYTHON_OUT_DIR/__init__.py" ]; then
    echo -e "${YELLOW}创建 Python __init__.py...${NC}"
    cat > "$PYTHON_OUT_DIR/__init__.py" <<EOF
# gRPC module
from .server import AIServicer, create_grpc_server

__all__ = [
    "AIServicer",
    "create_grpc_server",
]
EOF
    echo -e "${GREEN}✓ __init__.py 已创建${NC}"
fi

echo ""

# ============== 完成 ==============

echo -e "${GREEN}=== 代码生成完成 ===${NC}"
echo ""
echo "生成文件列表:"
echo "  Go:"
ls "$GO_OUT_DIR"/*.go 2>/dev/null | awk '{print "    " $1}'
echo ""
echo "  Python:"
ls "$PYTHON_OUT_DIR"/ai_pb2* 2>/dev/null | awk '{print "    " $1}'
echo ""
echo -e "${GREEN}下一步:${NC}"
echo "  1. Go 服务端: 在 server/main.go 中使用生成的代码"
echo "  2. Python 服务端: 在 model/app/grpc/server.py 中实现服务"
echo ""
