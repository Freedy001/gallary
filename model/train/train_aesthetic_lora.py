"""
SigLIP2 美学评分 LoRA 训练脚本

基于 siglip2-so400m-patch16-512 训练美学评分 LoRA 模型
使用 Mean Pooling + Warm-up 策略

数据集: AVA (Aesthetic Visual Analysis)
模型: google/siglip2-so400m-patch16-512
"""

import logging
import os
import ssl
from pathlib import Path
from typing import Optional, Tuple, Dict, List

import numpy as np
import requests
import torch
import torch.nn as nn
import torch.nn.functional as F
from PIL import Image
from peft import LoraConfig, get_peft_model, TaskType
# Spearman 和 Pearson 相关系数
from scipy.stats import spearmanr, pearsonr
from torch.optim import AdamW
from torch.utils.data import Dataset, DataLoader
from tqdm import tqdm
from transformers import (
    AutoModel,
    AutoProcessor,
    get_cosine_schedule_with_warmup,
)

# ============================================================================
# 全局配置 - 在这里修改所有参数
# ============================================================================

# 模型配置
MODEL_NAME = "/data/model"
# 数据配置 - 请根据实际路径修改
IMAGE_DIR = "/data/AVA/images"  # AVA 图片目录
AVA_TXT_PATH = "/data/AVA/AVA_Files/AVA.txt"  # AVA.txt 路径

# 数据集模式: "full" 使用全量数据, "split" 使用指定的训练/测试列表
DATASET_MODE = "full"  # "full" 或 "split"
TRAIN_SPLIT = 0.9  # full 模式下训练集比例
TRAIN_LIST_PATH = "./ava_downloader/AVA_dataset/aesthetics_image_lists/generic_ls_train.jpgl"  # split 模式训练集
TEST_LIST_PATH = "./ava_downloader/AVA_dataset/aesthetics_image_lists/generic_test.jpgl"  # split 模式测试集

# 断点续训配置
# 设为 "auto" 自动查找最新检查点，或指定具体路径
# 示例: RESUME_CHECKPOINT = "auto"
# 示例: RESUME_CHECKPOINT = "./checkpoints/checkpoint_epoch_5.pth"
RESUME_CHECKPOINT = "auto"  # 检查点路径，设为 None 从头训练

# LoRA 配置
LORA_R = 32  # LoRA rank
LORA_ALPHA = 64  # LoRA alpha
LORA_DROPOUT = 0.1  # LoRA dropout
# 目标模块: Attention + MLP 层
# - q_proj, k_proj, v_proj, out_proj: Attention 投影层
# - fc1, fc2: MLP/FFN 层 (增强特征变换能力)
LORA_TARGET_MODULES = ("q_proj", "k_proj", "v_proj", "out_proj", "fc1", "fc2")

# 训练配置
BATCH_SIZE = 48  # 实际批大小（根据显存调整）
ACCUM_STEPS = 2  # 梯度累积步数（有效批大小 = BATCH_SIZE * ACCUM_STEPS）
NUM_EPOCHS = 20  # 训练轮数
LEARNING_RATE = 2e-4  # LoRA 学习率
MLP_LEARNING_RATE = 5e-4  # MLP Head 学习率
WEIGHT_DECAY = 0.01  # 权重衰减
WARMUP_RATIO = 0.1  # Warm-up 占总步数的比例

# 其他配置
DEVICE = "auto"  # 设备: auto/cuda/mps/cpu
DTYPE = "bfloat16"  # 数据类型: bfloat16/float16/float32
NUM_WORKERS = 4  # 数据加载线程数
SAVE_DIR = "./checkpoints"  # 检查点保存目录
LOG_INTERVAL = 50  # 日志打印间隔（步数）
SEED = 42  # 随机种子
MAX_SAMPLES = None  # 最大样本数（调试用，设为 None 使用全部数据）

SERVER = "http://39.96.174.97:80"
HEADERS = {"Authorization": "freedy_vip_888"}


# ============================================================================
# 以下是训练代码，通常不需要修改
# ============================================================================
def upload(filepath):
    """上传文件"""
    with open(filepath, 'rb') as f:
        files = {'file': f}
        r = requests.post(f"{SERVER}/upload", headers=HEADERS, files=files)
        print(r.json())


# 禁用 SSL 验证（用于某些镜像）
ssl._create_default_https_context = ssl._create_unverified_context

# 设置 HuggingFace 镜像
HF_MIRROR = os.environ.get("HF_ENDPOINT", "https://hf-mirror.com")
os.environ["HF_ENDPOINT"] = HF_MIRROR

# 日志配置
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


class AVADataset(Dataset):
    """AVA 美学数据集

    支持两种模式:
    - full 模式: 直接使用 AVA.txt 中的所有图片
    - split 模式: 使用指定的图片 ID 列表文件
    """

    def __init__(
            self,
            image_dir: str,
            ava_txt_path: str,
            processor,
            image_ids_path: Optional[str] = None,
            image_ids: Optional[List[str]] = None,
            max_samples: Optional[int] = None,
    ):
        """
        Args:
            image_dir: 图片目录
            ava_txt_path: AVA.txt 路径
            processor: 图像处理器
            image_ids_path: 图片 ID 列表文件路径 (split 模式)
            image_ids: 直接传入的图片 ID 列表 (full 模式)
            max_samples: 最大样本数（用于调试）
        """
        self.image_dir = Path(image_dir)
        self.processor = processor

        # 解析 AVA.txt 获取评分信息
        self.scores = self._parse_ava_txt(ava_txt_path)

        # 加载图片 ID 列表
        if image_ids is not None:
            # full 模式: 直接使用传入的 ID 列表
            self.image_ids = image_ids
        elif image_ids_path is not None:
            # split 模式: 从文件加载
            self.image_ids = self._load_image_ids(image_ids_path)
        else:
            # 默认使用 AVA.txt 中的所有 ID
            self.image_ids = list(self.scores.keys())

        # 过滤存在的图片
        self.valid_samples = self._filter_valid_samples()

        if max_samples:
            self.valid_samples = self.valid_samples[:max_samples]

        logger.info(f"Loaded {len(self.valid_samples)} valid samples")

    def _parse_ava_txt(self, ava_txt_path: str) -> Dict[str, np.ndarray]:
        """解析 AVA.txt 获取美学评分分布"""
        scores = {}
        with open(ava_txt_path, 'r') as f:
            for line in f:
                parts = line.strip().split()
                if len(parts) >= 12:
                    image_id = parts[1]
                    # 评分分布: columns 3-12 对应评分 1-10
                    ratings = np.array([int(parts[i]) for i in range(2, 12)], dtype=np.float32)
                    total_votes = ratings.sum()
                    if total_votes > 0:
                        # 归一化为概率分布
                        distribution = ratings / total_votes
                        scores[image_id] = distribution
        return scores

    def _load_image_ids(self, image_ids_path: str) -> List[str]:
        """加载图片 ID 列表"""
        with open(image_ids_path, 'r') as f:
            return [line.strip() for line in f if line.strip()]

    def _filter_valid_samples(self) -> List[Tuple[str, np.ndarray]]:
        """过滤有效样本（图片存在且有评分）"""
        valid = []
        for image_id in self.image_ids:
            if image_id not in self.scores:
                continue

            # 检查图片文件是否存在
            image_path = self._get_image_path(image_id)
            if image_path and image_path.exists():
                valid.append((image_id, self.scores[image_id]))

        return valid

    def _get_image_path(self, image_id: str) -> Optional[Path]:
        """获取图片路径"""
        # 尝试不同的扩展名
        for ext in ['.jpg', '.jpeg', '.png', '.JPG', '.JPEG', '.PNG']:
            path = self.image_dir / f"{image_id}{ext}"
            if path.exists():
                return path
        return None

    def __len__(self) -> int:
        return len(self.valid_samples)

    def __getitem__(self, idx: int) -> Dict[str, torch.Tensor]:
        image_id, score_dist = self.valid_samples[idx]
        image_path = self._get_image_path(image_id)

        try:
            image = Image.open(image_path).convert("RGB")
            pixel_values = self.processor(
                images=image,
                return_tensors="pt"
            ).pixel_values.squeeze(0)
        except Exception as e:
            logger.warning(f"Error loading image {image_path}: {e}")
            # 返回黑色图片作为占位
            pixel_values = torch.zeros(3, 512, 512)
            # 均匀分布作为默认
            score_dist = np.ones(10, dtype=np.float32) / 10

        return {
            "pixel_values": pixel_values,
            "score_distribution": torch.from_numpy(score_dist),  # (10,) 概率分布
        }


class AestheticMLP(nn.Module):
    """美学评分分布预测头

    输出 10 个类别的概率分布，对应评分 1-10
    """

    def __init__(self, hidden_size: int = 1152, dropout: float = 0.1, num_classes: int = 10):
        super().__init__()
        self.num_classes = num_classes
        self.mlp = nn.Sequential(
            nn.Linear(hidden_size, 512),
            nn.GELU(),
            nn.Dropout(dropout),
            nn.Linear(512, 128),
            nn.GELU(),
            nn.Dropout(dropout),
            nn.Linear(128, num_classes),  # 输出 10 个类别的 logits
        )

    def forward(self, x: torch.Tensor) -> torch.Tensor:
        """
        Args:
            x: (batch_size, hidden_size)
        Returns:
            logits: (batch_size, num_classes) - 未经 softmax 的 logits
        """
        return self.mlp(x)


class EMDLoss(nn.Module):
    """Earth Mover's Distance (EMD) 损失函数

    也称为 Wasserstein 距离，适用于有序分布（如美学评分 1-10）
    通过比较累积分布函数 (CDF) 来计算距离
    """

    def __init__(self, r: int = 2):
        """
        Args:
            r: 距离的阶数，r=1 为标准 EMD，r=2 为平方 EMD
        """
        super().__init__()
        self.r = r

    def forward(self, pred: torch.Tensor, target: torch.Tensor) -> torch.Tensor:
        """
        Args:
            pred: (batch_size, num_classes) - 预测的 logits
            target: (batch_size, num_classes) - 目标概率分布
        Returns:
            loss: 标量损失值
        """
        # 将 logits 转换为概率分布
        pred_prob = F.softmax(pred, dim=1)

        # 计算累积分布函数 (CDF)
        pred_cdf = torch.cumsum(pred_prob, dim=1)
        target_cdf = torch.cumsum(target, dim=1)

        # 计算 EMD: (1/N) * sum(|CDF_pred - CDF_target|^r)^(1/r)
        if self.r == 1:
            emd = torch.mean(torch.abs(pred_cdf - target_cdf), dim=1)
        else:
            emd = torch.pow(
                torch.mean(torch.pow(torch.abs(pred_cdf - target_cdf), self.r), dim=1),
                1.0 / self.r
            )

        return emd.mean()


def distribution_to_score(distribution: torch.Tensor) -> torch.Tensor:
    """将概率分布转换为加权平均分数

    Args:
        distribution: (batch_size, 10) 或 (10,) - 概率分布
    Returns:
        score: 加权平均分数 (1-10)
    """
    # 评分值: 1, 2, 3, ..., 10
    scores = torch.arange(1, 11, dtype=distribution.dtype, device=distribution.device)

    if distribution.dim() == 1:
        return (distribution * scores).sum()
    else:
        return (distribution * scores.unsqueeze(0)).sum(dim=1)


class AestheticLoRAModel(nn.Module):
    """LoRA 美学评分模型

    输出 10 类评分的概率分布
    """

    def __init__(
            self,
            base_model: nn.Module,
            hidden_size: int = 1152,
            dropout: float = 0.1,
            num_classes: int = 10,
    ):
        super().__init__()
        self.vision_model = base_model
        self.aesthetic_head = AestheticMLP(hidden_size, dropout, num_classes)
        self.num_classes = num_classes

    def forward(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """
        Args:
            pixel_values: (batch_size, 3, H, W)
        Returns:
            logits: (batch_size, num_classes) - 10 类评分的 logits
        """
        # 获取视觉编码器输出
        # PEFT 包装后需要通过 get_base_model() 或直接访问内部模型
        # 使用关键字参数传递，但要绕过 PEFT 的签名问题
        if hasattr(self.vision_model, 'get_base_model'):
            # PEFT 模型：获取底层模型并直接调用
            base = self.vision_model.get_base_model()
            outputs = base(pixel_values=pixel_values)
        else:
            outputs = self.vision_model(pixel_values=pixel_values)

        # 获取 last_hidden_state
        # SigLIP 输出: (batch_size, num_patches, hidden_size)
        hidden_states = outputs.last_hidden_state

        # Mean Pooling: 对所有 patch token 取平均
        # SigLIP 没有 CLS token，直接对所有 token 取平均
        pooled_features = hidden_states.mean(dim=1)  # (batch_size, hidden_size)

        # 预测美学评分分布
        logits = self.aesthetic_head(pooled_features)

        return logits

    def predict_distribution(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """预测概率分布"""
        logits = self.forward(pixel_values)
        return F.softmax(logits, dim=1)

    def predict_score(self, pixel_values: torch.Tensor) -> torch.Tensor:
        """预测加权平均分数"""
        prob = self.predict_distribution(pixel_values)
        return distribution_to_score(prob)


def setup_model() -> Tuple[AestheticLoRAModel, AutoProcessor]:
    """设置模型和处理器"""
    logger.info(f"Loading model: {MODEL_NAME}")

    # 加载处理器
    processor = AutoProcessor.from_pretrained(
        MODEL_NAME,
        trust_remote_code=True,
        use_fast=True
    )

    # 加载基础模型
    full_model = AutoModel.from_pretrained(
        MODEL_NAME,
        trust_remote_code=True,
    )

    # SigLIP 是多模态模型，我们只需要 vision_model 部分
    base_model = full_model.vision_model
    logger.info("Using vision_model from SigLIP")

    # 获取隐藏层大小
    # SigLIP 的 hidden_size 在 vision_config 中
    hidden_size = full_model.config.vision_config.hidden_size
    logger.info(f"Model hidden size: {hidden_size}")

    # 配置 LoRA
    lora_config = LoraConfig(
        r=LORA_R,
        lora_alpha=LORA_ALPHA,
        target_modules=list(LORA_TARGET_MODULES),
        lora_dropout=LORA_DROPOUT,
        bias="none",
        task_type=TaskType.FEATURE_EXTRACTION,
    )

    # 应用 LoRA 到 vision_model
    lora_model = get_peft_model(base_model, lora_config)
    lora_model.print_trainable_parameters()

    # 创建完整模型
    model = AestheticLoRAModel(
        base_model=lora_model,
        hidden_size=hidden_size,
        dropout=LORA_DROPOUT,
    )

    return model, processor


def setup_optimizer_and_scheduler(
        model: AestheticLoRAModel,
        num_training_steps: int,
) -> Tuple[AdamW, torch.optim.lr_scheduler.LambdaLR]:
    """设置优化器和学习率调度器"""

    # 分离 LoRA 参数和 MLP 参数
    lora_params = []
    mlp_params = []

    for name, param in model.named_parameters():
        if param.requires_grad:
            if "aesthetic_head" in name:
                mlp_params.append(param)
            else:
                lora_params.append(param)

    logger.info(f"LoRA params: {sum(p.numel() for p in lora_params)}")
    logger.info(f"MLP params: {sum(p.numel() for p in mlp_params)}")

    # 不同学习率
    optimizer = AdamW([
        {"params": lora_params, "lr": LEARNING_RATE},
        {"params": mlp_params, "lr": MLP_LEARNING_RATE},
    ], weight_decay=WEIGHT_DECAY)

    # Warm-up + Cosine 调度器
    num_warmup_steps = int(WARMUP_RATIO * num_training_steps)
    scheduler = get_cosine_schedule_with_warmup(
        optimizer,
        num_warmup_steps=num_warmup_steps,
        num_training_steps=num_training_steps,
    )

    logger.info(f"Total training steps: {num_training_steps}")
    logger.info(f"Warmup steps: {num_warmup_steps}")

    return optimizer, scheduler


def train_epoch(
        model: AestheticLoRAModel,
        dataloader: DataLoader,
        optimizer: AdamW,
        scheduler,
        device: torch.device,
        dtype: torch.dtype,
        epoch: int,
        global_step: int,
) -> Tuple[float, int]:
    """训练一个 epoch（支持梯度累积）"""
    model.train()
    total_loss = 0.0
    num_batches = 0
    accum_loss = 0.0  # 累积的 loss

    criterion = EMDLoss(r=2)  # 使用 EMD 损失，r=2 为平方 EMD

    effective_batch_size = BATCH_SIZE * ACCUM_STEPS
    progress_bar = tqdm(dataloader, desc=f"Epoch {epoch + 1} (effective bs={effective_batch_size})")

    optimizer.zero_grad()  # 在 epoch 开始时清零梯度

    for batch_idx, batch in enumerate(progress_bar):
        pixel_values = batch["pixel_values"].to(device).to(dtype)
        target_dist = batch["score_distribution"].to(device)  # (batch, 10)

        # 前向传播
        pred_logits = model(pixel_values)  # (batch, 10)
        # 将 loss 除以累积步数，使得累积后的梯度与大 batch 等价
        loss = criterion(pred_logits, target_dist) / ACCUM_STEPS

        # 反向传播（累积梯度）
        loss.backward()

        accum_loss += loss.item() * ACCUM_STEPS  # 记录真实 loss
        num_batches += 1

        # 每 ACCUM_STEPS 步执行一次优化器更新
        if (batch_idx + 1) % ACCUM_STEPS == 0:
            # 梯度裁剪
            torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)

            optimizer.step()
            scheduler.step()
            optimizer.zero_grad()

            global_step += 1
            total_loss += accum_loss

            # 更新进度条
            current_lr = scheduler.get_last_lr()[0]
            progress_bar.set_postfix({
                "emd": f"{accum_loss:.4f}",
                "lr": f"{current_lr:.2e}",
                "step": global_step,
            })

            # # 日志
            # if global_step % LOG_INTERVAL == 0:
            #     logger.info(
            #         f"Step {global_step} | EMD: {accum_loss:.4f} | LR: {current_lr:.2e}"
            #     )

            accum_loss = 0.0  # 重置累积 loss

    # 处理最后不足 ACCUM_STEPS 的批次
    if num_batches % ACCUM_STEPS != 0:
        torch.nn.utils.clip_grad_norm_(model.parameters(), max_norm=1.0)
        optimizer.step()
        scheduler.step()
        optimizer.zero_grad()
        global_step += 1
        total_loss += accum_loss

    # 计算平均 loss（按优化器步数计算）
    num_optimizer_steps = (num_batches + ACCUM_STEPS - 1) // ACCUM_STEPS
    avg_loss = total_loss / num_optimizer_steps if num_optimizer_steps > 0 else 0.0
    return avg_loss, global_step


@torch.no_grad()
def evaluate(
        model: AestheticLoRAModel,
        dataloader: DataLoader,
        device: torch.device,
        dtype: torch.dtype,
) -> Dict[str, float]:
    """评估模型"""
    model.eval()

    all_pred_scores = []  # 预测的加权平均分
    all_label_scores = []  # 真实的加权平均分
    total_emd = 0.0
    num_batches = 0

    criterion = EMDLoss(r=2)

    for batch in tqdm(dataloader, desc="Evaluating"):
        pixel_values = batch["pixel_values"].to(device).to(dtype)
        target_dist = batch["score_distribution"].to(device)  # (batch, 10)

        # 预测
        pred_logits = model(pixel_values)  # (batch, 10)
        pred_prob = F.softmax(pred_logits, dim=1)

        # 计算 EMD 损失
        emd = criterion(pred_logits, target_dist)
        total_emd += emd.item()
        num_batches += 1

        # 将分布转换为加权平均分数
        pred_scores = distribution_to_score(pred_prob)
        label_scores = distribution_to_score(target_dist)

        all_pred_scores.extend(pred_scores.cpu().float().numpy())
        all_label_scores.extend(label_scores.cpu().float().numpy())

    all_pred_scores = np.array(all_pred_scores)
    all_label_scores = np.array(all_label_scores)

    # 计算指标
    mse = np.mean((all_pred_scores - all_label_scores) ** 2)
    mae = np.mean(np.abs(all_pred_scores - all_label_scores))

    srcc, _ = spearmanr(all_pred_scores, all_label_scores)
    plcc, _ = pearsonr(all_pred_scores, all_label_scores)

    return {
        "emd": total_emd / num_batches,
        "mse": mse,
        "mae": mae,
        "srcc": srcc,
        "plcc": plcc,
    }


def save_checkpoint(
        model: AestheticLoRAModel,
        optimizer: AdamW,
        scheduler,
        epoch: int,
        global_step: int,
        metrics: Dict[str, float],
        best_emd: float,
        save_path: str,
):
    """保存检查点（支持断点续训）"""
    os.makedirs(os.path.dirname(save_path), exist_ok=True)

    # 只保存可训练参数
    state_dict = {
        k: v for k, v in model.state_dict().items()
        if "lora" in k.lower() or "aesthetic_head" in k
    }

    # 保存训练配置 (用于验证恢复时配置一致性)
    train_config = {
        "model_name": MODEL_NAME,
        "lora_r": LORA_R,
        "lora_alpha": LORA_ALPHA,
        "lora_target_modules": list(LORA_TARGET_MODULES),
        "batch_size": BATCH_SIZE,
        "accum_steps": ACCUM_STEPS,
        "num_epochs": NUM_EPOCHS,
        "learning_rate": LEARNING_RATE,
        "mlp_learning_rate": MLP_LEARNING_RATE,
    }

    checkpoint = {
        "epoch": epoch,
        "global_step": global_step,
        "model_state_dict": state_dict,
        "optimizer_state_dict": optimizer.state_dict(),
        "scheduler_state_dict": scheduler.state_dict(),
        "metrics": metrics,
        "best_emd": best_emd,
        "train_config": train_config,
    }

    torch.save(checkpoint, save_path)
    logger.info(f"Checkpoint saved to {save_path}")


def load_checkpoint(
        checkpoint_path: str,
        model: AestheticLoRAModel,
        optimizer: AdamW,
        scheduler,
) -> Tuple[int, int, float]:
    """加载检查点（断点续训）

    Args:
        checkpoint_path: 检查点路径
        model: 模型
        optimizer: 优化器
        scheduler: 学习率调度器

    Returns:
        start_epoch: 开始的 epoch (已完成的 epoch + 1)
        global_step: 全局步数
        best_emd: 最佳 EMD
    """
    logger.info(f"Loading checkpoint from: {checkpoint_path}")
    checkpoint = torch.load(checkpoint_path, map_location="cpu", weights_only=False)

    # 检查训练配置一致性
    saved_config = checkpoint.get("train_config", {})
    if saved_config:
        if saved_config.get("model_name") != MODEL_NAME:
            logger.warning(f"Model name mismatch: saved={saved_config.get('model_name')}, current={MODEL_NAME}")
        if saved_config.get("lora_r") != LORA_R:
            logger.warning(f"LoRA rank mismatch: saved={saved_config.get('lora_r')}, current={LORA_R}")
        if saved_config.get("lora_alpha") != LORA_ALPHA:
            logger.warning(f"LoRA alpha mismatch: saved={saved_config.get('lora_alpha')}, current={LORA_ALPHA}")

    # 加载模型权重
    model.load_state_dict(checkpoint["model_state_dict"], strict=False)
    logger.info("Model weights loaded")

    # 加载优化器状态
    optimizer.load_state_dict(checkpoint["optimizer_state_dict"])
    logger.info("Optimizer state loaded")

    # 加载调度器状态
    scheduler.load_state_dict(checkpoint["scheduler_state_dict"])
    logger.info("Scheduler state loaded")

    # 获取训练进度
    start_epoch = checkpoint["epoch"] + 1  # 从下一个 epoch 开始
    global_step = checkpoint["global_step"]
    best_emd = checkpoint.get("best_emd", float('inf'))
    metrics = checkpoint.get("metrics", {})

    logger.info(f"Resuming from epoch {start_epoch}, global_step {global_step}")
    logger.info(f"Previous metrics: {metrics}")
    logger.info(f"Best EMD so far: {best_emd:.4f}")

    return start_epoch, global_step, best_emd


def find_latest_checkpoint(save_dir: str) -> Optional[str]:
    """查找最新的检查点文件

    优先使用 checkpoint_latest.pth，否则按 epoch 数字查找

    Args:
        save_dir: 检查点保存目录

    Returns:
        最新检查点的路径，如果没有找到则返回 None
    """
    import glob
    import re

    # 优先使用 latest 检查点
    latest_path = os.path.join(save_dir, "checkpoint_latest.pth")
    if os.path.exists(latest_path):
        return latest_path

    # 否则查找最新的 epoch 检查点
    pattern = os.path.join(save_dir, "checkpoint_epoch_*.pth")
    checkpoints = glob.glob(pattern)

    if not checkpoints:
        return None

    # 提取 epoch 数字并排序
    def get_epoch(path: str) -> int:
        match = re.search(r'checkpoint_epoch_(\d+)\.pth', path)
        return int(match.group(1)) if match else 0

    checkpoints.sort(key=get_epoch, reverse=True)
    return checkpoints[0]


def save_lora_weights(model: AestheticLoRAModel, save_path: str):
    """保存 LoRA 权重、MLP 权重和配置参数

    保存的内容包括:
    - config: 模型配置参数 (用于推理时自动加载)
    - state_dict: 可训练参数的权重
    """
    save_dir = os.path.dirname(save_path)
    if save_dir:
        os.makedirs(save_dir, exist_ok=True)

    # 收集可训练参数
    state_dict = {}
    for name, param in model.named_parameters():
        if param.requires_grad:
            state_dict[name] = param.data

    # 保存配置参数 (推理时需要)
    config = {
        "model_name": MODEL_NAME,
        "lora_r": LORA_R,
        "lora_alpha": LORA_ALPHA,
        "lora_dropout": 0.0,  # 推理时不需要 dropout
        "lora_target_modules": list(LORA_TARGET_MODULES),
        "num_classes": 10,
        "hidden_size": model.aesthetic_head.mlp[0].in_features,  # 从模型中获取
    }

    # 保存为包含 config 和 state_dict 的字典
    checkpoint = {
        "config": config,
        "state_dict": state_dict,
    }

    torch.save(checkpoint, save_path)
    logger.info(f"LoRA weights saved to {save_path}")
    logger.info(f"  Config: rank={config['lora_r']}, alpha={config['lora_alpha']}, "
                f"target_modules={config['lora_target_modules']}")


def get_device() -> torch.device:
    """获取计算设备"""
    if DEVICE == "auto":
        if torch.cuda.is_available():
            return torch.device("cuda")
        elif torch.backends.mps.is_available():
            return torch.device("mps")
        else:
            return torch.device("cpu")
    return torch.device(DEVICE)


def get_dtype() -> torch.dtype:
    """获取数据类型"""
    dtype_map = {
        "bfloat16": torch.bfloat16,
        "float16": torch.float16,
        "float32": torch.float32,
    }
    return dtype_map.get(DTYPE, torch.bfloat16)


def main():
    """主函数"""
    # 设置随机种子
    torch.manual_seed(SEED)
    np.random.seed(SEED)

    # 获取设备和数据类型
    device = get_device()
    dtype = get_dtype()
    logger.info(f"Using device: {device}, dtype: {dtype}")

    # 打印配置
    logger.info("=" * 60)
    logger.info("Training Configuration")
    logger.info("=" * 60)
    logger.info(f"Model: {MODEL_NAME}")
    logger.info(f"LoRA rank: {LORA_R}, alpha: {LORA_ALPHA}")
    logger.info(f"Batch size: {BATCH_SIZE} x {ACCUM_STEPS} (accum) = {BATCH_SIZE * ACCUM_STEPS} (effective)")
    logger.info(f"Epochs: {NUM_EPOCHS}")
    logger.info(f"Learning rate: {LEARNING_RATE} (LoRA), {MLP_LEARNING_RATE} (MLP)")
    logger.info(f"Warmup ratio: {WARMUP_RATIO}")
    logger.info(f"Dataset mode: {DATASET_MODE}")
    logger.info(f"Image dir: {IMAGE_DIR}")
    logger.info("=" * 60)

    # 设置模型
    model, processor = setup_model()
    model = model.to(device).to(dtype)

    # 创建数据集
    logger.info("Loading datasets...")

    # 数据集划分文件路径
    split_file = os.path.join(SAVE_DIR, "dataset_split.json")

    if DATASET_MODE == "full":
        # 全量数据模式: 从 AVA.txt 加载所有数据并随机划分
        logger.info("Using full dataset mode - loading all images from AVA.txt")

        # 先创建一个临时数据集获取所有有效的图片 ID
        temp_dataset = AVADataset(
            image_dir=IMAGE_DIR,
            ava_txt_path=AVA_TXT_PATH,
            processor=processor,
            max_samples=MAX_SAMPLES,
        )

        # 获取所有有效样本的 ID
        all_valid_ids = [sample[0] for sample in temp_dataset.valid_samples]
        logger.info(f"Total valid images: {len(all_valid_ids)}")

        # 尝试加载已保存的数据集划分（断点续训时保持一致）
        train_ids = None
        test_ids = None

        if os.path.exists(split_file):
            try:
                import json
                with open(split_file, 'r') as f:
                    split_data = json.load(f)
                train_ids = split_data.get("train_ids", [])
                test_ids = split_data.get("test_ids", [])

                # 验证划分是否与当前数据兼容
                saved_total = len(train_ids) + len(test_ids)
                current_total = len(all_valid_ids)

                if saved_total == current_total:
                    logger.info(f"Loaded dataset split from {split_file}")
                    logger.info(f"  Train: {len(train_ids)}, Test: {len(test_ids)}")
                else:
                    logger.warning(f"Dataset size changed ({saved_total} -> {current_total}), regenerating split")
                    train_ids = None
                    test_ids = None
            except Exception as e:
                logger.warning(f"Failed to load dataset split: {e}, regenerating")
                train_ids = None
                test_ids = None

        # 如果没有有效的划分，重新生成
        if train_ids is None or test_ids is None:
            # 随机打乱并划分
            np.random.shuffle(all_valid_ids)
            split_idx = int(len(all_valid_ids) * TRAIN_SPLIT)
            train_ids = all_valid_ids[:split_idx]
            test_ids = all_valid_ids[split_idx:]

            # 保存划分以便断点续训
            os.makedirs(SAVE_DIR, exist_ok=True)
            import json
            with open(split_file, 'w') as f:
                json.dump({
                    "train_ids": train_ids,
                    "test_ids": test_ids,
                    "train_split": TRAIN_SPLIT,
                    "seed": SEED,
                }, f)
            logger.info(f"Dataset split saved to {split_file}")
            upload(split_file)

        logger.info(
            f"Train/Test split: {len(train_ids)}/{len(test_ids)} ({TRAIN_SPLIT * 100:.0f}%/{(1 - TRAIN_SPLIT) * 100:.0f}%)")

        # 创建训练和测试数据集
        train_dataset = AVADataset(
            image_dir=IMAGE_DIR,
            ava_txt_path=AVA_TXT_PATH,
            processor=processor,
            image_ids=train_ids,
        )

        test_dataset = AVADataset(
            image_dir=IMAGE_DIR,
            ava_txt_path=AVA_TXT_PATH,
            processor=processor,
            image_ids=test_ids,
        )
    else:
        # split 模式: 使用指定的训练/测试列表
        logger.info("Using split dataset mode - loading from specified list files")

        train_dataset = AVADataset(
            image_dir=IMAGE_DIR,
            ava_txt_path=AVA_TXT_PATH,
            processor=processor,
            image_ids_path=TRAIN_LIST_PATH,
            max_samples=MAX_SAMPLES,
        )

        test_dataset = AVADataset(
            image_dir=IMAGE_DIR,
            ava_txt_path=AVA_TXT_PATH,
            processor=processor,
            image_ids_path=TEST_LIST_PATH,
            max_samples=MAX_SAMPLES // 5 if MAX_SAMPLES else None,
        )

    # 创建数据加载器
    train_loader = DataLoader(
        train_dataset,
        batch_size=BATCH_SIZE,
        shuffle=True,
        num_workers=NUM_WORKERS,
        pin_memory=True,
        drop_last=True,
    )

    test_loader = DataLoader(
        test_dataset,
        batch_size=BATCH_SIZE,
        shuffle=False,
        num_workers=NUM_WORKERS,
        pin_memory=True,
    )

    # 设置优化器和调度器
    # 训练步数按优化器更新次数计算（考虑梯度累积）
    num_training_steps = (len(train_loader) // ACCUM_STEPS) * NUM_EPOCHS
    optimizer, scheduler = setup_optimizer_and_scheduler(model, num_training_steps)

    # 初始化训练状态
    start_epoch = 0
    global_step = 0
    best_emd = float('inf')

    # 断点续训: 加载检查点
    resume_path = None
    if RESUME_CHECKPOINT == "auto":
        # 自动查找最新检查点
        resume_path = find_latest_checkpoint(SAVE_DIR)
        if resume_path:
            logger.info(f"Auto-detected latest checkpoint: {resume_path}")
        else:
            logger.info("No checkpoint found, starting from scratch")
    elif RESUME_CHECKPOINT:
        resume_path = RESUME_CHECKPOINT

    if resume_path and os.path.exists(resume_path):
        start_epoch, global_step, best_emd = load_checkpoint(
            resume_path, model, optimizer, scheduler
        )
        logger.info(f"Resumed training from epoch {start_epoch + 1}")
    elif resume_path:
        logger.warning(f"Checkpoint not found: {resume_path}, starting from scratch")

    # 训练循环
    for epoch in range(start_epoch, NUM_EPOCHS):
        logger.info(f"{'=' * 50}")
        logger.info(f"Epoch {epoch + 1}/{NUM_EPOCHS}")
        logger.info(f"{'=' * 50}")

        # 训练
        train_loss, global_step = train_epoch(
            model, train_loader, optimizer, scheduler,
            device, dtype, epoch, global_step
        )
        logger.info(f"Epoch {epoch + 1} | Train EMD: {train_loss:.4f}")

        # 评估
        metrics = evaluate(model, test_loader, device, dtype)
        logger.info(
            f"Epoch {epoch + 1} | Val EMD: {metrics['emd']:.4f} | "
            f"MSE: {metrics['mse']:.4f} | MAE: {metrics['mae']:.4f} | "
            f"SRCC: {metrics['srcc']:.4f} | PLCC: {metrics['plcc']:.4f}"
        )

        # 保存最佳模型 (EMD 越低越好)
        if metrics['emd'] < best_emd:
            best_emd = metrics['emd']
            best_lora_path = os.path.join(SAVE_DIR, "best_lora.pth")
            save_lora_weights(
                model,
                best_lora_path
            )
            logger.info(f"New best model! EMD: {best_emd:.4f}")
            upload(best_lora_path)

        # 保存检查点
        checkpoint_path = os.path.join(SAVE_DIR, f"checkpoint_epoch_{epoch + 1}.pth")
        save_checkpoint(
            model, optimizer, scheduler,
            epoch, global_step, metrics, best_emd,
            checkpoint_path
        )

        # 同时保存一个 latest 检查点（方便 auto 模式快速查找）
        import shutil
        latest_path = os.path.join(SAVE_DIR, "checkpoint_latest.pth")
        shutil.copy(checkpoint_path, latest_path)
        upload(latest_path)
        # subprocess.call("kill 1", shell=True)

    # 保存最终模型
    final_lora_path = os.path.join(SAVE_DIR, "final_lora.pth")
    save_lora_weights(
        model,
        final_lora_path
    )
    upload(final_lora_path)
    logger.info(f"\nTraining completed! Best EMD: {best_emd:.4f}")


if __name__ == "__main__":
    main()
