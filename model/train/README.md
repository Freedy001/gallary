# 美学评分 LoRA 模型

基于 SigLIP2 视觉模型的美学评分 LoRA 微调模型，使用 AVA 数据集训练。

## 模型架构

- **基础模型**: `google/siglip2-so400m-patch16-512`
- **微调方法**: LoRA (Low-Rank Adaptation)
- **输出**: 1-10 分的概率分布
- **评分计算**: 加权平均分

## 文件说明

| 文件 | 说明 |
|------|------|
| `train_aesthetic_lora.py` | 训练脚本 |
| `inference_aesthetic_lora.py` | 推理脚本 |
| `requirements.txt` | 依赖包 |

## 训练

### 配置

修改 `train_aesthetic_lora.py` 顶部的配置：

```python
# 模型配置
MODEL_NAME = "google/siglip2-so400m-patch16-512"

# 数据配置
IMAGE_DIR = "/path/to/ava_images"  # AVA 图片目录
AVA_TXT_PATH = "./AVA.txt"         # AVA.txt 路径

# LoRA 配置
LORA_R = 16          # LoRA rank
LORA_ALPHA = 32      # LoRA alpha
LORA_TARGET_MODULES = ("q_proj", "k_proj", "v_proj", "out_proj", "fc1", "fc2")

# 训练配置
BATCH_SIZE = 48      # 批大小
ACCUM_STEPS = 2      # 梯度累积
NUM_EPOCHS = 20      # 训练轮数
LEARNING_RATE = 2e-4 # LoRA 学习率

# 断点续训
RESUME_CHECKPOINT = "auto"  # "auto" 或 具体路径 或 None
```

### 运行训练

```bash
cd model
pip install -r requirements.txt
python train_aesthetic_lora.py
```

### 输出文件

训练完成后，在 `./checkpoints/` 目录下生成：

| 文件 | 说明 |
|------|------|
| `best_lora.pth` | 最佳模型权重 (SRCC 最高) |
| `final_lora.pth` | 最终模型权重 |
| `checkpoint_epoch_*.pth` | 各 epoch 检查点 |
| `checkpoint_latest.pth` | 最新检查点 (用于断点续训) |
| `dataset_split.json` | 数据集划分 (保证断点续训一致性) |

## 推理

### 命令行使用

```bash
python inference_aesthetic_lora.py image1.jpg image2.jpg \
    --lora_weights ./checkpoints/best_lora.pth \
    --show_distribution
```

### Python API

```python
from inference_aesthetic_lora import AestheticPredictor

# 初始化预测器 (配置自动从权重文件读取)
predictor = AestheticPredictor(
    lora_weights_path="./checkpoints/best_lora.pth",
    device="auto",  # auto/cuda/mps/cpu
    dtype="bfloat16",
)

# 单张图片预测
score, distribution = predictor.predict("image.jpg")
print(f"美学评分: {score:.2f}")
print(f"评分分布: {distribution}")

# 批量预测
scores, distributions = predictor.predict_batch(["img1.jpg", "img2.jpg"])
```

## 评分等级

| 分数范围 | 等级 |
|---------|------|
| >= 7.5 | 优秀 (Excellent) |
| >= 6.5 | 很好 (Very Good) |
| >= 5.5 | 良好 (Good) |
| >= 4.5 | 一般 (Average) |
| >= 3.5 | 较差 (Below Average) |
| < 3.5 | 差 (Poor) |

## 技术细节

### 损失函数

使用 **EMD (Earth Mover's Distance)** 损失函数，适用于有序分布：

```
EMD = mean(|CDF_pred - CDF_target|^r)^(1/r)
```

其中 r=2 (平方 EMD)。

### 评估指标

- **EMD**: Earth Mover's Distance (越低越好)
- **MSE**: 均方误差
- **MAE**: 平均绝对误差
- **SRCC**: Spearman 秩相关系数 (越高越好)
- **PLCC**: Pearson 线性相关系数 (越高越好)

### 特征提取

- 使用 SigLIP vision_model 提取图像特征
- Mean Pooling: 对所有 patch token 取平均
- MLP Head: 1152 -> 512 -> 128 -> 10

## 断点续训

支持三种模式：

```python
# 从头训练
RESUME_CHECKPOINT = None

# 自动查找最新检查点
RESUME_CHECKPOINT = "auto"

# 指定检查点
RESUME_CHECKPOINT = "./checkpoints/checkpoint_epoch_5.pth"
```

断点续训会：
1. 恢复模型权重
2. 恢复优化器状态
3. 恢复学习率调度器
4. 恢复 best_srcc 记录
5. 使用相同的数据集划分 (从 `dataset_split.json` 加载)

## 参考

- [AVA Dataset](https://github.com/imfing/ava_downloader) - Aesthetic Visual Analysis
- [SigLIP](https://huggingface.co/google/siglip2-so400m-patch16-512) - Google Vision-Language Model
- [PEFT](https://github.com/huggingface/peft) - Parameter-Efficient Fine-Tuning
- [NIMA](https://arxiv.org/abs/1709.05424) - Neural Image Assessment
