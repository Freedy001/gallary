# -*- coding: utf-8 -*-
"""
LoRA 美学评分推理脚本 (ONNX Runtime 版本)

使用 ONNX Runtime 进行快速 CPU 推理
需要先使用 export_onnx.py 导出 ONNX 模型

使用方法:
1. 先导出 ONNX 模型:
   python export_onnx.py --weights ./best_lora.pth --output ./model.onnx

2. 使用 ONNX 推理:
   python inference_aesthetic_onnx.py --onnx ./model.onnx image.jpg
"""
import argparse
import os
import ssl
import time
from typing import List, Tuple

import numpy as np
from PIL import Image
from torch.utils.data import Dataset, DataLoader

from model import (
    distribution_to_score_numpy,
    softmax_numpy,
    get_score_level,
    format_distribution,
)

ssl._create_default_https_context = ssl._create_unverified_context

HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR

# 导入 ONNX Runtime
try:
    import onnxruntime as ort

    ONNX_AVAILABLE = True
except ImportError:
    ONNX_AVAILABLE = False
    print("警告: onnxruntime 未安装，请运行: pip install onnxruntime")


class ImageDataset(Dataset):
    """图片数据集，用于批量加载

    支持 PyTorch DataLoader 或普通批量处理
    """

    def __init__(self, image_paths: List[str], processor):
        super().__init__()
        self.image_paths = image_paths
        self.processor = processor

    def __len__(self):
        return len(self.image_paths)

    def __getitem__(self, idx):
        image_path = self.image_paths[idx]
        try:
            image = Image.open(image_path).convert("RGB")
            # 预处理图片,先获取 PyTorch 张量再转换为 numpy
            pixel_values = self.processor(
                images=image,
                return_tensors="pt"
            ).pixel_values.numpy().astype(np.float32)[0]  # 移除batch维度

            return pixel_values, image_path, True, None
        except Exception as e:
            # 返回错误标记和错误信息
            return np.zeros((3, 224, 224), dtype=np.float32), image_path, False, str(e)


def collate_fn(batch):
    """自定义 collate 函数,用于 DataLoader

    将 numpy 数组转换为批次
    """
    pixels, paths, valids, errors = zip(*batch)
    # 转换为 numpy 数组
    pixels_array = np.stack(pixels, axis=0)
    return pixels_array, list(paths), list(valids), list(errors)


class ONNXPredictor:
    """ONNX Runtime 推理后端

    使用 ONNX Runtime 进行快速 CPU 推理
    """

    def __init__(self, onnx_path: str, num_threads: int = 4):
        if not ONNX_AVAILABLE:
            raise RuntimeError("onnxruntime 未安装，请运行: pip install onnxruntime")

        if not os.path.exists(onnx_path):
            raise FileNotFoundError(f"ONNX 模型文件不存在: {onnx_path}")

        # 配置 session options
        sess_options = ort.SessionOptions()
        sess_options.intra_op_num_threads = num_threads
        sess_options.inter_op_num_threads = num_threads
        sess_options.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL

        # 创建推理 session
        self.session = ort.InferenceSession(
            onnx_path,
            sess_options,
            providers=['CPUExecutionProvider']
        )

        # 获取输入输出名称
        self.input_name = self.session.get_inputs()[0].name
        self.output_name = self.session.get_outputs()[0].name

        print(f"ONNX model loaded: {onnx_path}")
        print(f"  Input: {self.input_name}")
        print(f"  Output: {self.output_name}")
        print(f"  Threads: {num_threads}")

    def predict(self, pixel_values: np.ndarray) -> np.ndarray:
        """运行推理，返回 logits"""
        outputs = self.session.run(
            [self.output_name],
            {self.input_name: pixel_values}
        )
        return outputs[0]

    def predict_distribution(self, pixel_values: np.ndarray) -> np.ndarray:
        """预测概率分布"""
        logits = self.predict(pixel_values)
        return softmax_numpy(logits, axis=-1)


class AestheticONNXPredictor:
    """美学评分预测器 (ONNX Runtime 版本)

    使用 ONNX Runtime 进行快速 CPU 推理
    需要提供:
    - ONNX 模型文件路径
    - 处理器来源 (可以是 PyTorch 权重文件路径或模型名称)
    """

    def __init__(
            self,
            onnx_path: str,
            base_model: str,
            num_threads: int = 4,
    ):
        self.num_threads = num_threads

        # 加载 ONNX 模型
        self.onnx_predictor = ONNXPredictor(onnx_path, num_threads)

        # 加载处理器
        from transformers import AutoProcessor
        print(f"Loading processor from: {base_model}")
        self.processor = AutoProcessor.from_pretrained(
            base_model,
            trust_remote_code=True,
            use_fast=True
        )

    def _preprocess(self, image_path: str) -> np.ndarray:
        """预处理图片，返回 float32 numpy 数组"""
        image = Image.open(image_path).convert("RGB")
        pixel_values = self.processor(
            images=image,
            return_tensors="pt"
        ).pixel_values.numpy().astype(np.float32)
        return pixel_values

    def _preprocess_batch(self, image_paths: List[str]) -> np.ndarray:
        """批量预处理图片"""
        images = [Image.open(p).convert("RGB") for p in image_paths]
        pixel_values = self.processor(
            images=images,
            return_tensors="pt"
        ).pixel_values.numpy().astype(np.float32)
        return pixel_values

    def predict(self, image_path: str) -> Tuple[float, np.ndarray]:
        """预测单张图片的美学评分

        Returns:
            score: 加权平均分数 (1-10)
            distribution: 10 类概率分布
        """
        pixel_values = self._preprocess(image_path)
        distribution = self.onnx_predictor.predict_distribution(pixel_values)
        score = distribution_to_score_numpy(distribution.squeeze(0))
        return float(score), distribution.squeeze(0)

    def predict_batch(self, image_paths: List[str]) -> Tuple[List[float], np.ndarray]:
        """批量预测

        Returns:
            scores: 加权平均分数列表
            distributions: (batch, 10) 概率分布
        """
        pixel_values = self._preprocess_batch(image_paths)
        distributions = self.onnx_predictor.predict_distribution(pixel_values)
        scores = [distribution_to_score_numpy(d) for d in distributions]
        return scores, distributions

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
    parser = argparse.ArgumentParser(description="Aesthetic score prediction (ONNX Runtime)")
    parser.add_argument("images", nargs="*", help="图片路径")
    parser.add_argument("--onnx", type=str, default="./model.onnx", help="ONNX 模型路径")
    parser.add_argument("--base_model", type=str, default="../siglip2", help="处理器来源 (权重文件路径或模型名称)")
    parser.add_argument("--show_distribution", action="store_true", help="显示评分概率分布", default=True)
    parser.add_argument("--threads", type=int, default=4, help="CPU 线程数")
    parser.add_argument("--batch_size", type=int, default=1, help="批处理大小")
    parser.add_argument("--num_workers", type=int, default=2, help="预处理工作进程数 (0表示不使用多进程)")

    args = parser.parse_args()

    if not ONNX_AVAILABLE:
        print("错误: onnxruntime 未安装")
        print("请运行: pip install onnxruntime")
        return

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
    predictor = AestheticONNXPredictor(
        onnx_path=args.onnx,
        base_model=args.base_model,
        num_threads=args.threads,
    )

    results = []
    start_time = time.time()

    # 使用 DataLoader 批量推理 (如果有 PyTorch)
    if args.batch_size > 1 and len(image_files) > 1:
        print(f"使用 DataLoader: batch_size={args.batch_size}, num_workers={args.num_workers}\n")

        # 创建数据集和数据加载器
        dataset = ImageDataset(image_files, predictor.processor)
        dataloader = DataLoader(
            dataset,
            batch_size=args.batch_size,
            num_workers=args.num_workers,
            shuffle=False,
            collate_fn=collate_fn,
            persistent_workers=args.num_workers > 0
        )

        processed = 0
        for batch_pixels, batch_paths, batch_valid, batch_errors in dataloader:
            # 过滤掉加载失败的图片
            valid_indices = [i for i, v in enumerate(batch_valid) if v]
            if not valid_indices:
                for i, path in enumerate(batch_paths):
                    error_msg = batch_errors[i] if batch_errors[i] else "未知错误"
                    print(f"[{processed + 1}/{len(image_files)}] {path}: 加载失败 - {error_msg}")
                    processed += 1
                continue

            # 只处理有效的图片
            valid_pixels = batch_pixels[valid_indices]
            valid_paths = [batch_paths[i] for i in valid_indices]

            try:
                # 批量推理
                distributions = predictor.onnx_predictor.predict_distribution(valid_pixels)
                scores = [distribution_to_score_numpy(d) for d in distributions]

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
                inner_time = time.time()
                score, distribution = predictor.predict(image_path)
                level = get_score_level(score)
                results.append((image_path, score, level, distribution))
                print(f"[{i}/{len(image_files)}] {image_path}")
                print(f"  分数: {score:.2f} - {level}")
                if args.show_distribution:
                    print("  评分分布:")
                    print(format_distribution(distribution))
                print(f"耗时: {time.time() - inner_time:.2f} 秒")
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
