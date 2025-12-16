"""
Aesthetic Predictor V2.5 Demo
基于 SigLIP 的改进版图片美学评分模型

模型来源: https://github.com/discus0434/aesthetic-predictor-v2-5
评分范围: 1-10 (5.5+ 被认为是高美学分数)
特点: 支持更广泛的图像类型，包括插画、动漫等
"""

import os
import ssl

ssl._create_default_https_context = ssl._create_unverified_context
from pathlib import Path

import torch
from aesthetic_predictor_v2_5 import convert_v2_5_from_siglip
from PIL import Image

HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR
print(f"Using HuggingFace mirror: {HF_MIRROR}")


class AestheticPredictorV25:
    """Aesthetic Predictor V2.5 封装类"""

    def __init__(self, device: str = None):
        """
        初始化美学评估器

        Args:
            device: 运行设备 ('cuda', 'cpu', 'mps')，None 则自动选择
        """
        # 自动选择设备
        if device is None:
            if torch.cuda.is_available():
                device = "cuda"
            elif torch.backends.mps.is_available():
                device = "mps"
            else:
                device = "cpu"
        self.device = device
        self._load_models()

    def _load_models(self):
        # load model and preprocessor
        model, preprocessor = convert_v2_5_from_siglip(
            encoder_model_name="/Users/wuyuejiang/.cache/huggingface/hub/models--google--siglip-so400m-patch14-384/snapshots/9fdffc58afc957d1a03a25b10dba0329ab15c2a3",
            low_cpu_mem_usage=True,
            trust_remote_code=True,
        )
        self.preprocessor = preprocessor
        self.model = model.to(torch.bfloat16).to(self.device)
        print("Models loaded successfully!")

    def predict(self, image_path: str) -> float:
        """
        预测单张图片的美学评分

        Args:
            image_path: 图片路径

        Returns:
            美学评分 (1-10)
        """
        # load image to evaluate
        image = Image.open(image_path).convert("RGB")

        # preprocess image
        pixel_values = (
            self.preprocessor(images=image, return_tensors="pt")
            .pixel_values.to(torch.bfloat16)
            .to(self.device)
        )

        # predict aesthetic score
        with torch.inference_mode():
            score = self.model(pixel_values).logits.squeeze().float().cpu().numpy()

        # print result
        print(f"Aesthetics score: {score:.2f}")
        return score


def get_score_level_v25(score: float) -> str:
    """
    根据 V2.5 分数返回等级描述
    注意: V2.5 的评分标准与 V2 不同，5.5+ 被认为是高美学分数
    """
    if score >= 7.5:
        return "优秀 (Excellent)"
    elif score >= 6.5:
        return "很好 (Very Good)"
    elif score >= 5.5:
        return "良好 (Good)"
    elif score >= 4.5:
        return "一般 (Average)"
    elif score >= 3.5:
        return "较差 (Below Average)"
    else:
        return "差 (Poor)"


def demo():
    """演示如何使用 Aesthetic Predictor V2.5"""
    print("=" * 60)
    print("Aesthetic Predictor V2.5 Demo")
    print("=" * 60)

    # 初始化预测器
    predictor = AestheticPredictorV25()

    # 查找测试图片
    test_dirs = [
        "/Volumes/ALL-PIC",
    ]

    test_images = []
    for test_dir in test_dirs:
        if os.path.exists(test_dir):
            for ext in ['*.jpg', '*.jpeg', '*.png', '*.webp']:
                test_images.extend(Path(test_dir).rglob(ext))

    if not test_images:
        print("\n未找到测试图片。请提供图片路径：")
        print("用法: python demo_aesthetic_v2_5.py <image_path>")
        print("\n或者将图片放入以下目录之一：")
        for d in test_dirs:
            print(f"  - {d}")
        return

    # 测试前 5 张图片
    print(f"\n找到 {len(test_images)} 张图片，测试前 5 张：\n")

    results = []
    for img_path in list(test_images):
        try:
            score = predictor.predict(str(img_path))
            level = get_score_level_v25(score)
            results.append((str(img_path), score, level))
            print(f"  {img_path.name}")
            print(f"    Score: {score:.2f} - {level}")
            print()
        except Exception as e:
            print(f"  {img_path.name}")
            print(f"    Error: {e}")
            print()

    # 按分数排序
    if results:
        print("-" * 60)
        print("按美学评分排序 (V2.5: 5.5+ 为高分):")
        print("-" * 60)
        for path, score, level in sorted(results, key=lambda x: x[1], reverse=True):
            print(f"  {score:.2f} - {path}")


if __name__ == "__main__":
    import sys

    if len(sys.argv) > 1:
        # 命令行指定图片
        predictor = AestheticPredictorV25()
        for img_path in sys.argv[1:]:
            if os.path.exists(img_path):
                score = predictor.predict(img_path)
                level = get_score_level_v25(score)
                print(f"{img_path}: {score:.2f} - {level}")
            else:
                print(f"{img_path}: 文件不存在")
    else:
        demo()
