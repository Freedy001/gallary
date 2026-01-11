"""
Image Aesthetic & Embedding Service
基于 SigLIP2 + 自训练 LoRA 的图片美学评分与向量嵌入微服务

启动方式:
    python main.py                              # 默认 PyTorch 后端
    BACKEND=onnx python main.py                 # 使用 ONNX 后端
    GRPC_PORT=50051 python main.py              # 自定义 gRPC 端口
"""

import os
import signal
import sys

import torch

from app.grpc import create_grpc_server
from app.services import model_service, BackendType

# gRPC 服务器实例
grpc_server = None


def initialize_and_start():
    """初始化模型并启动 gRPC 服务"""
    global grpc_server

    # 获取配置
    device = os.environ.get("DEVICE", None)
    if device is None:
        if torch.cuda.is_available():
            device = "cuda"
        elif torch.backends.mps.is_available():
            device = "mps"
        else:
            device = "cpu"

    backend_str = os.environ.get("BACKEND", "pytorch").lower()
    backend = BackendType.ONNX if backend_str == "onnx" else BackendType.PYTORCH

    # 模型路径配置
    base_model_path = os.environ.get("BASE_MODEL_PATH", None)
    lora_weights_path = os.environ.get("LORA_WEIGHTS_PATH", None)
    onnx_model_path = os.environ.get("ONNX_MODEL_PATH", None)

    print(f"Initializing model with backend: {backend_str}, device: {device}")
    
    # 初始化模型
    model_service.initialize(
        device=device,
        backend=backend,
        base_model_path=base_model_path,
        lora_weights_path=lora_weights_path,
        onnx_model_path=onnx_model_path,
    )

    # 启动 gRPC 服务器
    grpc_port = int(os.environ.get("GRPC_PORT", 50051))
    grpc_server = create_grpc_server(port=grpc_port)
    grpc_server.start()
    print(f"✅ gRPC server started successfully on port {grpc_port}")
    print(f"Model loaded on device: {device}")


def shutdown_handler(_signum, _frame):
    """处理关闭信号"""
    global grpc_server
    print("\n收到关闭信号，正在停止 gRPC 服务...")
    if grpc_server:
        grpc_server.stop(grace=5)
        print("gRPC server stopped")
    sys.exit(0)


if __name__ == "__main__":
    # 注册信号处理
    signal.signal(signal.SIGINT, shutdown_handler)
    signal.signal(signal.SIGTERM, shutdown_handler)

    print("="*60)
    print("Image Aesthetic & Embedding Service (gRPC Only)")
    print("基于 SigLIP2 + 自训练 LoRA 的图片美学评分与向量嵌入服务")
    print("="*60)
    
    # 初始化并启动服务
    initialize_and_start()
    
    print("\n服务运行中，按 Ctrl+C 停止...\n")
    
    # 保持主线程运行
    try:
        signal.pause()
    except AttributeError:
        # Windows 不支持 signal.pause()
        import time
        while True:
            time.sleep(1)
