# -*- coding: utf-8 -*-
"""
导出 ONNX 模型

将 PyTorch 模型导出为 ONNX 格式，用于 ONNX Runtime 快速推理

支持导出:
1. 美学评分模型 (LoRA + MLP head)
2. SigLIP 图像嵌入模型 (用于图文搜索)
3. SigLIP 文本嵌入模型 (用于图文搜索)

使用方法:
    # 导出所有模型
    python export_onnx.py --all --model ../siglip2

    # 仅导出美学评分模型
    python export_onnx.py --weights ./best_lora.pth --output ./model.onnx

    # 仅导出 SigLIP 嵌入模型
    python export_onnx.py --export-siglip --model ../siglip2
"""
import argparse
import os
import warnings
from typing import Tuple

import torch
import torch.nn as nn

# 复用模型定义和预测器
from inference_aesthetic_lora import AestheticPredictor


class ONNXWrapper(nn.Module):
    """ONNX 导出包装器

    将 AestheticLoRAModel 包装为更简单的结构，避免 ONNX 导出时的兼容性问题
    """

    def __init__(self, model: nn.Module):
        super().__init__()
        self.model = model

    def forward(self, pixel_values: torch.Tensor) -> torch.Tensor:
        return self.model(pixel_values)


class SigLIPVisionWrapper(nn.Module):
    """SigLIP 视觉模型 ONNX 导出包装器

    输出归一化的图像嵌入向量
    注意：SigLIP 的 pooler_output 已经是最终嵌入，无需额外投影层
    """

    def __init__(self, siglip_model: nn.Module):
        super().__init__()
        self.vision_model = siglip_model.vision_model

    def forward(self, pixel_values: torch.Tensor) -> torch.Tensor:
        vision_outputs = self.vision_model(pixel_values=pixel_values)
        image_features = vision_outputs.pooler_output
        # 归一化
        image_features = image_features / image_features.norm(dim=-1, keepdim=True)
        return image_features


class SigLIPTextWrapper(nn.Module):
    """SigLIP 文本模型 ONNX 导出包装器

    输出归一化的文本嵌入向量
    """

    def __init__(self, siglip_model: nn.Module):
        super().__init__()
        self.text_model = siglip_model.text_model
        self.text_projection = siglip_model.text_projection

    def forward(self, input_ids: torch.Tensor, attention_mask: torch.Tensor) -> torch.Tensor:
        text_outputs = self.text_model(
            input_ids=input_ids,
            attention_mask=attention_mask,
        )
        pooled_output = text_outputs.pooler_output
        text_features = self.text_projection(pooled_output)
        # 归一化
        text_features = text_features / text_features.norm(dim=-1, keepdim=True)
        return text_features


def verify_and_test_onnx(output_path: str, test_inputs: dict, expected_output_shape: Tuple[int, ...]):
    """验证并测试 ONNX 模型"""
    # 验证导出的模型
    try:
        import onnx
        print("\n验证 ONNX 模型...")
        onnx_model = onnx.load(output_path)
        onnx.checker.check_model(onnx_model)
        print("ONNX 模型验证通过!")

        print(f"\n模型信息:")
        print(f"  IR 版本: {onnx_model.ir_version}")
        print(f"  Opset 版本: {onnx_model.opset_import[0].version}")
        print(f"  生产者: {onnx_model.producer_name}")

    except ImportError:
        print("onnx 未安装，跳过验证")
        print("安装: pip install onnx")
    except Exception as e:
        print(f"验证警告: {e}")

    # 测试 ONNX Runtime
    try:
        import onnxruntime as ort
        import numpy as np

        print("\n测试 ONNX Runtime 推理...")
        session = ort.InferenceSession(output_path, providers=['CPUExecutionProvider'])

        # 转换输入为 numpy
        numpy_inputs = {k: v.numpy() for k, v in test_inputs.items()}
        outputs = session.run(None, numpy_inputs)

        print(f"  输出形状: {outputs[0].shape}")
        print("ONNX Runtime 推理测试通过!")

    except ImportError:
        print("\nonnxruntime 未安装，跳过推理测试")
        print("安装: pip install onnxruntime")
    except Exception as e:
        print(f"推理测试失败: {e}")

    # 打印文件大小
    file_size = os.path.getsize(output_path) / (1024 * 1024)
    print(f"\n文件大小: {file_size:.2f} MB")


def export_aesthetic_onnx(
        predictor: AestheticPredictor,
        output_path: str,
        image_size: Tuple[int, int] = (512, 512),
        opset_version: int = 17,
):
    """导出美学评分模型为 ONNX 格式

    Args:
        predictor: AestheticPredictor 实例
        output_path: ONNX 文件保存路径
        image_size: 输入图片尺寸 (height, width)
        opset_version: ONNX opset 版本
    """
    print(f"\n导出美学评分 ONNX 模型: {output_path}")
    print(f"  输入尺寸: {image_size}")
    print(f"  ONNX opset: {opset_version}")

    # 创建 dummy input (使用 float32)
    dummy_input = torch.randn(1, 3, image_size[0], image_size[1], dtype=torch.float32)

    # 确保模型在 CPU 上且为 float32
    model = predictor.model.cpu().float()
    model.eval()

    # 包装模型
    wrapped_model = ONNXWrapper(model)
    wrapped_model.eval()

    # 使用旧版 ONNX 导出 API (更兼容)
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        torch.onnx.export(
            wrapped_model,
            (dummy_input,),
            output_path,
            export_params=True,
            opset_version=opset_version,
            do_constant_folding=True,
            input_names=['pixel_values'],
            output_names=['logits'],
            dynamic_axes={
                'pixel_values': {0: 'batch_size'},
                'logits': {0: 'batch_size'}
            },
            dynamo=False,
        )

    print("美学评分 ONNX 模型导出成功!")

    # 验证和测试
    verify_and_test_onnx(
        output_path,
        {'pixel_values': dummy_input},
        (1, 10),
    )

    return output_path


def export_siglip_vision_onnx(
        base_model_path: str,
        output_path: str,
        image_size: Tuple[int, int] = (512, 512),
        opset_version: int = 17,
):
    """导出 SigLIP 视觉模型为 ONNX 格式

    用于图像嵌入向量提取

    Args:
        base_model_path: SigLIP 基础模型路径
        output_path: ONNX 文件保存路径
        image_size: 输入图片尺寸 (height, width)
        opset_version: ONNX opset 版本
    """
    from transformers import SiglipModel

    print(f"\n导出 SigLIP 视觉 ONNX 模型: {output_path}")
    print(f"  基础模型: {base_model_path}")
    print(f"  输入尺寸: {image_size}")
    print(f"  ONNX opset: {opset_version}")

    # 加载 SigLIP 模型
    print("  加载 SigLIP 模型...")
    siglip_model = SiglipModel.from_pretrained(base_model_path)
    siglip_model = siglip_model.float().cpu()
    siglip_model.eval()

    # 包装模型
    wrapped_model = SigLIPVisionWrapper(siglip_model)
    wrapped_model.eval()

    # 创建 dummy input
    dummy_input = torch.randn(1, 3, image_size[0], image_size[1], dtype=torch.float32)

    # 导出
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        torch.onnx.export(
            wrapped_model,
            (dummy_input,),
            output_path,
            export_params=True,
            opset_version=opset_version,
            do_constant_folding=True,
            input_names=['pixel_values'],
            output_names=['image_embeds'],
            dynamic_axes={
                'pixel_values': {0: 'batch_size'},
                'image_embeds': {0: 'batch_size'}
            },
            dynamo=False,
        )

    print("SigLIP 视觉 ONNX 模型导出成功!")

    # 验证和测试
    verify_and_test_onnx(
        output_path,
        {'pixel_values': dummy_input},
        (1, 1152),
    )

    return output_path


def export_siglip_text_onnx(
        base_model_path: str,
        output_path: str,
        max_length: int = 64,
        opset_version: int = 17,
):
    """导出 SigLIP 文本模型为 ONNX 格式

    用于文本嵌入向量提取

    Args:
        base_model_path: SigLIP 基础模型路径
        output_path: ONNX 文件保存路径
        max_length: 最大文本长度
        opset_version: ONNX opset 版本
    """
    from transformers import SiglipModel

    print(f"\n导出 SigLIP 文本 ONNX 模型: {output_path}")
    print(f"  基础模型: {base_model_path}")
    print(f"  最大长度: {max_length}")
    print(f"  ONNX opset: {opset_version}")

    # 加载 SigLIP 模型
    print("  加载 SigLIP 模型...")
    siglip_model = SiglipModel.from_pretrained(base_model_path)
    siglip_model = siglip_model.float().cpu()
    siglip_model.eval()

    # 包装模型
    wrapped_model = SigLIPTextWrapper(siglip_model)
    wrapped_model.eval()

    # 创建 dummy input
    dummy_input_ids = torch.randint(0, 32000, (1, max_length), dtype=torch.long)
    dummy_attention_mask = torch.ones(1, max_length, dtype=torch.long)

    # 导出
    with warnings.catch_warnings():
        warnings.simplefilter("ignore")
        torch.onnx.export(
            wrapped_model,
            (dummy_input_ids, dummy_attention_mask),
            output_path,
            export_params=True,
            opset_version=opset_version,
            do_constant_folding=True,
            input_names=['input_ids', 'attention_mask'],
            output_names=['text_embeds'],
            dynamic_axes={
                'input_ids': {0: 'batch_size', 1: 'sequence_length'},
                'attention_mask': {0: 'batch_size', 1: 'sequence_length'},
                'text_embeds': {0: 'batch_size'}
            },
            dynamo=False,
        )

    print("SigLIP 文本 ONNX 模型导出成功!")

    # 验证和测试
    verify_and_test_onnx(
        output_path,
        {'input_ids': dummy_input_ids, 'attention_mask': dummy_attention_mask},
        (1, 1152),
    )

    return output_path


def main():
    parser = argparse.ArgumentParser(
        description="导出 ONNX 模型",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
示例:
  # 导出所有模型 (美学评分 + SigLIP 嵌入)
  python export_onnx.py --all --model ../siglip2 --weights ./best_lora.pth

  # 仅导出美学评分模型
  python export_onnx.py --weights ./best_lora.pth --output ./model.onnx

  # 仅导出 SigLIP 嵌入模型 (视觉 + 文本)
  python export_onnx.py --export-siglip --model ../siglip2

导出后的模型:
  - model.onnx: 美学评分模型
  - siglip_vision.onnx: 图像嵌入模型
  - siglip_text.onnx: 文本嵌入模型
        """
    )

    parser.add_argument("--weights", type=str, default="./best_lora.pth",
                        help="PyTorch 权重文件路径 (美学评分模型)")
    parser.add_argument("--model", type=str, default=None,
                        help="基础模型路径 (默认: ../siglip2)")
    parser.add_argument("--output", type=str, default="./model.onnx",
                        help="美学评分 ONNX 输出路径")
    parser.add_argument("--output-dir", type=str, default=".",
                        help="ONNX 模型输出目录")
    parser.add_argument("--opset", type=int, default=17,
                        help="ONNX opset 版本 (推荐 17)")

    # 导出模式选项
    parser.add_argument("--all", action="store_true",
                        help="导出所有模型 (美学评分 + SigLIP 嵌入)")
    parser.add_argument("--export-siglip", action="store_true",
                        help="导出 SigLIP 嵌入模型 (视觉 + 文本)")
    parser.add_argument("--export-aesthetic", action="store_true",
                        help="导出美学评分模型")
    parser.add_argument("--export-vision", action="store_true",
                        help="仅导出 SigLIP 视觉模型")
    parser.add_argument("--export-text", action="store_true",
                        help="仅导出 SigLIP 文本模型")

    args = parser.parse_args()

    # 设置默认模型路径
    if args.model is None:
        args.model = "../siglip2"

    # 创建输出目录
    if args.output_dir and not os.path.exists(args.output_dir):
        os.makedirs(args.output_dir)

    # 确定导出哪些模型
    export_aesthetic = args.export_aesthetic or args.all
    export_vision = args.export_vision or args.export_siglip or args.all
    export_text = args.export_text or args.export_siglip or args.all

    # 如果没有指定任何导出选项，默认导出美学评分模型
    if not any([args.all, args.export_siglip, args.export_aesthetic,
                args.export_vision, args.export_text]):
        export_aesthetic = True

    image_size = (512, 512)

    # 导出美学评分模型
    if export_aesthetic:
        if not os.path.exists(args.weights):
            print(f"警告: 权重文件不存在: {args.weights}")
            print("跳过美学评分模型导出")
        else:
            print("\n加载美学评分模型...")
            predictor = AestheticPredictor(
                lora_weights_path=args.weights,
                model_name=args.model,
                device="cpu",
                dtype="float32",
            )

            aesthetic_output = os.path.join(args.output_dir, "model.onnx")
            export_aesthetic_onnx(
                predictor=predictor,
                output_path=aesthetic_output,
                image_size=image_size,
                opset_version=args.opset,
            )

    # 导出 SigLIP 视觉模型
    if export_vision:
        if not os.path.exists(args.model):
            print(f"错误: 基础模型不存在: {args.model}")
        else:
            vision_output = os.path.join(args.output_dir, "siglip_vision.onnx")
            export_siglip_vision_onnx(
                base_model_path=args.model,
                output_path=vision_output,
                image_size=image_size,
                opset_version=args.opset,
            )

    # 导出 SigLIP 文本模型
    if export_text:
        if not os.path.exists(args.model):
            print(f"错误: 基础模型不存在: {args.model}")
        else:
            text_output = os.path.join(args.output_dir, "siglip_text.onnx")
            export_siglip_text_onnx(
                base_model_path=args.model,
                output_path=text_output,
                opset_version=args.opset,
            )

    print("\n" + "=" * 50)
    print("导出完成!")
    print("=" * 50)

    # 打印使用说明
    print(f"\n导出的模型位于: {os.path.abspath(args.output_dir)}")
    if export_aesthetic:
        print(f"  - model.onnx: 美学评分模型")
    if export_vision:
        print(f"  - siglip_vision.onnx: 图像嵌入模型")
    if export_text:
        print(f"  - siglip_text.onnx: 文本嵌入模型")


if __name__ == "__main__":
    main()
