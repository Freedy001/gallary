# -*- coding: utf-8 -*-
"""
LoRA 美学评分推理脚本 (PyTorch 版本)

加载训练好的 LoRA 权重进行推理
输出 1-10 分的概率分布，并计算加权平均分

使用 float32 进行 CPU 推理以获得更好的性能
"""
import argparse
import os
import ssl
import time
from typing import List, Tuple

import numpy as np
import torch
from PIL import Image
from peft import LoraConfig, get_peft_model, TaskType
from torch.utils.data import Dataset, DataLoader
from transformers import AutoModel, AutoProcessor

from model import (
    AestheticLoRAModel,
    distribution_to_score_torch,
    get_score_level,
    format_distribution,
)

ssl._create_default_https_context = ssl._create_unverified_context

HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR


class ImageDataset(Dataset):
    """图片数据集，用于批量加载

    支持多进程预加载图片
    """

    def __init__(self, image_paths: List[str], processor):
        self.image_paths = image_paths
        self.processor = processor

    def __len__(self):
        return len(self.image_paths)

    def __getitem__(self, idx):
        image_path = self.image_paths[idx]
        try:
            image = Image.open(image_path).convert("RGB")
            # 预处理图片
            pixel_values = self.processor(
                images=image,
                return_tensors="pt"
            ).pixel_values.squeeze(0)  # 移除batch维度
            return pixel_values, image_path, True
        except Exception as e:
            # 返回错误标记
            return torch.zeros(3, 224, 224), image_path, False


class AestheticPredictor:
    """美学评分预测器 (PyTorch 版本)

    自动从权重文件中读取配置参数，无需手动指定。
    支持两种格式的权重文件:
    - 新格式: {"config": {...}, "state_dict": {...}}
    - 旧格式: 直接保存的 state_dict
    """

    def __init__(
            self,
            lora_weights_path: str,
            model_name: str = None,  # 可选，优先使用权重文件中的配置
            device: str = "auto",
            dtype: str = "auto",  # auto: 根据设备自动选择最优类型
            num_threads: int = 4,  # CPU 线程数
    ):
        # 设置设备
        if device == "auto":
            if torch.cuda.is_available():
                self.device = torch.device("cuda")
            elif torch.backends.mps.is_available():
                self.device = torch.device("mps")
            else:
                self.device = torch.device("cpu")
        else:
            self.device = torch.device(device)

        # 设置数据类型 - 根据设备自动选择最优类型
        if dtype == "auto":
            self.dtype = self._get_optimal_dtype()
        else:
            dtype_map = {
                "bfloat16": torch.bfloat16,
                "float16": torch.float16,
                "float32": torch.float32,
            }
            self.dtype = dtype_map.get(dtype, torch.float32)

        # 设置 CPU 线程数
        if self.device.type == "cpu":
            torch.set_num_threads(num_threads)
            print(f"PyTorch threads: {num_threads}")

        # 加载模型
        self._load_model(lora_weights_path, model_name)

    def _get_optimal_dtype(self) -> torch.dtype:
        if self.device.type == "cuda":
            # 检查是否支持 bfloat16 (Ampere 架构, compute capability >= 8.0)
            if torch.cuda.is_bf16_supported():
                print("Auto dtype: bfloat16 (CUDA Ampere+)")
                return torch.bfloat16
            else:
                print("Auto dtype: float32 (CUDA)")
                return torch.float32
        elif self.device.type == "mps":
            # Apple Silicon 对 float16 有良好支持
            print("Auto dtype: bfloat16 (MPS)")
            return torch.bfloat16
        else:
            # CPU 推理使用 float32 更快更稳定
            print("Auto dtype: float32 (CPU)")
            return torch.float32

    def _load_model(self, lora_weights_path: str, model_name_override: str = None):
        # 加载权重文件
        if not lora_weights_path or not os.path.exists(lora_weights_path):
            raise ValueError(f"LoRA weights file not found: {lora_weights_path}")

        print(f"Loading LoRA weights from: {lora_weights_path}")
        checkpoint = torch.load(lora_weights_path, map_location="cpu")

        # 验证权重文件格式
        if not isinstance(checkpoint, dict) or "config" not in checkpoint or "state_dict" not in checkpoint:
            raise ValueError(
                f"Invalid weight file format. Expected dict with 'config' and 'state_dict' keys. "
                f"Got: {type(checkpoint).__name__}"
            )

        config = checkpoint["config"]
        state_dict = checkpoint["state_dict"]
        print(f"Loaded config from weights file:")
        print(f"  Model: {config['model_name']}")
        print(f"  LoRA rank: {config['lora_r']}, alpha: {config['lora_alpha']}")
        print(f"  Target modules: {config['lora_target_modules']}")

        # 允许通过参数覆盖模型名称
        model_name = model_name_override or config["model_name"]
        print(f"Loading base model: {model_name}")

        # 加载处理器
        self.processor = AutoProcessor.from_pretrained(
            model_name,
            trust_remote_code=True,
            use_fast=True
        )

        # 加载基础模型
        full_model = AutoModel.from_pretrained(
            model_name,
            trust_remote_code=True,
        )

        # SigLIP 是多模态模型，我们只需要 vision_model 部分
        if hasattr(full_model, 'vision_model'):
            base_model = full_model.vision_model
            print("Using vision_model from SigLIP")
        else:
            base_model = full_model

        # 获取隐藏层大小 (SigLIP 的 hidden_size 在 vision_config 中)
        if "hidden_size" in config:
            hidden_size = config["hidden_size"]
        elif hasattr(base_model.config, 'hidden_size'):
            hidden_size = base_model.config.hidden_size
        elif hasattr(full_model.config, 'vision_config'):
            hidden_size = full_model.config.vision_config.hidden_size
        else:
            hidden_size = 1152  # 默认值 (SigLIP-SO400M)

        # 配置 LoRA（从权重文件中读取）
        lora_config = LoraConfig(
            r=config["lora_r"],
            lora_alpha=config["lora_alpha"],
            target_modules=config["lora_target_modules"],
            lora_dropout=config.get("lora_dropout", 0.0),
            bias="none",
            task_type=TaskType.FEATURE_EXTRACTION,
        )

        lora_model = get_peft_model(base_model, lora_config)

        # 创建完整模型
        self.model = AestheticLoRAModel(
            base_model=lora_model,
            hidden_size=hidden_size,
            dropout=0.0,
            num_classes=config.get("num_classes", 10),
        )

        # 加载权重
        self.model.load_state_dict(state_dict, strict=False)

        self.model = self.model.to(self.device).to(self.dtype)
        self.model.eval()
        print(f"Model loaded successfully! (device={self.device}, dtype={self.dtype})")

    @torch.no_grad()
    def predict(self, image_path: str) -> Tuple[float, np.ndarray]:
        """预测单张图片的美学评分

        Returns:
            score: 加权平均分数 (1-10)
            distribution: 10 类概率分布
        """
        image = Image.open(image_path).convert("RGB")
        pixel_values = self.processor(
            images=image,
            return_tensors="pt"
        ).pixel_values.to(self.device).to(self.dtype)

        distribution = self.model.predict_distribution(pixel_values)
        score = distribution_to_score_torch(distribution).item()

        return score, distribution.squeeze(0).float().cpu().numpy()

    @torch.no_grad()
    def predict_batch(self, image_paths: List[str]) -> Tuple[List[float], np.ndarray]:
        """批量预测

        Returns:
            scores: 加权平均分数列表
            distributions: (batch, 10) 概率分布
        """
        images = [Image.open(p).convert("RGB") for p in image_paths]
        pixel_values = self.processor(
            images=images,
            return_tensors="pt"
        ).pixel_values.to(self.device).to(self.dtype)

        distributions = self.model.predict_distribution(pixel_values)
        scores = distribution_to_score_torch(distributions).cpu().numpy().tolist()

        return scores, distributions.float().cpu().numpy()

    @torch.no_grad()
    def predict_score_only(self, image_path: str) -> float:
        """只返回分数，不返回分布"""
        score, _ = self.predict(image_path)
        return score


def collect_image_files(paths: List[str]) -> List[str]:
    """收集图片文件路径

    Args:
        paths: 文件或目录路径列表
    Returns:
        所有图片文件的路径列表
    """
    image_extensions = {'.jpg', '.jpeg', '.png', '.bmp', '.gif', '.webp', '.tiff', '.tif'}
    image_files = []

    for path in paths:
        if os.path.isfile(path):
            # 如果是文件,直接添加
            image_files.append(path)
        elif os.path.isdir(path):
            # 如果是目录,递归扫描所有图片文件
            print(f"扫描目录: {path}")
            for root, _, files in os.walk(path):
                for file in files:
                    if os.path.splitext(file)[1].lower() in image_extensions:
                        image_files.append(os.path.join(root, file))
            print(f"  找到 {len([f for f in image_files if f.startswith(path)])} 个图片文件")
        else:
            print(f"{path}: 文件或目录不存在")

    return image_files


def main():
    parser = argparse.ArgumentParser(description="Aesthetic score prediction (PyTorch)")
    parser.add_argument("images", nargs="*", help="图片路径")
    parser.add_argument("--show_distribution", action="store_true", help="显示评分概率分布", default=True)
    parser.add_argument("--weights", type=str, default="./best_lora.pth", help="PyTorch 权重路径")
    parser.add_argument("--model", type=str, default="../siglip2", help="基础模型路径")
    parser.add_argument("--device", type=str, default="auto", help="设备 (auto/cpu/cuda/mps)")
    parser.add_argument("--dtype", type=str, default="auto", help="数据类型 (auto/float32/float16/bfloat16)")
    parser.add_argument("--threads", type=int, default=4, help="CPU 线程数")
    parser.add_argument("--batch_size", type=int, default=1, help="批处理大小")
    parser.add_argument("--num_workers", type=int, default=2, help="DataLoader 工作进程数")

    args = parser.parse_args()

    # 预测
    if not args.images:
        parser.print_help()
        return

    # 收集所有图片文件
    image_files = collect_image_files(args.images)
    if not image_files:
        print("未找到任何图片文件")
        return

    print(f"\n总共找到 {len(image_files)} 个图片文件\n")

    # 初始化预测器
    predictor = AestheticPredictor(
        lora_weights_path=args.weights,
        model_name=args.model,
        device=args.device,
        dtype=args.dtype,
        num_threads=args.threads,
    )

    results = []
    start_time = time.time()

    # 使用 DataLoader 批量推理
    if args.batch_size > 1 and len(image_files) > 1:
        print(f"使用 DataLoader: batch_size={args.batch_size}, num_workers={args.num_workers}\n")

        # 创建数据集和数据加载器
        dataset = ImageDataset(image_files, predictor.processor)
        dataloader = DataLoader(
            dataset,
            batch_size=args.batch_size,
            num_workers=args.num_workers,
            shuffle=False,
            pin_memory=predictor.device.type == "cuda",
            persistent_workers=args.num_workers > 0
        )

        processed = 0
        for batch_pixels, batch_paths, batch_valid in dataloader:
            # 过滤掉加载失败的图片
            valid_indices = [i for i, v in enumerate(batch_valid) if v]
            if not valid_indices:
                for path in batch_paths:
                    print(f"[{processed + 1}/{len(image_files)}] {path}: 加载失败")
                    processed += 1
                continue

            # 只处理有效的图片
            valid_pixels = batch_pixels[valid_indices].to(predictor.device).to(predictor.dtype)
            valid_paths = [batch_paths[i] for i in valid_indices]

            try:
                # 批量推理
                distributions = predictor.model.predict_distribution(valid_pixels)
                scores = distribution_to_score_torch(distributions).cpu().numpy()
                distributions = distributions.cpu().numpy()

                for path, score, dist in zip(valid_paths, scores, distributions):
                    level = get_score_level(score)
                    results.append((path, score, level, dist))
                    processed += 1
                    print(f"[{processed}/{len(image_files)}] {path}")
                    print(f"  分数: {score:.2f} - {level}")
                    if args.show_distribution:
                        print("  评分分布:")
                        print(format_distribution(dist))
            except Exception as e:
                print(f"批次处理失败: {e}")
                processed += len(valid_paths)
    else:
        # 单张推理
        for i, image_path in enumerate(image_files, 1):
            try:
                score, distribution = predictor.predict(image_path)
                level = get_score_level(score)
                results.append((image_path, score, level, distribution))
                print(f"[{i}/{len(image_files)}] {image_path}")
                print(f"  分数: {score:.2f} - {level}")
                if args.show_distribution:
                    print("  评分分布:")
                    print(format_distribution(distribution))
            except Exception as e:
                print(f"[{i}/{len(image_files)}] {image_path}: 处理失败 - {e}")

    # 输出汇总统计
    print("\n" + "=" * 50)
    print(f"处理完成: 成功 {len(results)}/{len(image_files)} 张图片")
    print(f"总耗时: {time.time() - start_time:.2f} 秒")
    if len(results) > 0:
        avg_score = sum(r[1] for r in results) / len(results)
        print(f"平均分数: {avg_score:.2f}")
    print("=" * 50)

    # 按分数排序
    if len(results) > 1:
        print("\n按美学评分排序:")
        for path, score, level, _ in sorted(results, key=lambda x: x[1], reverse=True):
            print(f"  {score:.2f} - {path}")


if __name__ == "__main__":
    main()
