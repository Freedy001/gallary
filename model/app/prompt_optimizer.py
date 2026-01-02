"""
Qwen3-0.6B 提示词优化服务
将中文搜索提示词优化为更适合 SigLIP 模型理解的英文描述

支持多种推理后端：
1. llama-cpp-python（推荐 CPU）：速度快 3-5 倍，使用 GGUF 格式
2. vLLM（推荐 GPU）：速度快 5-10 倍，仅支持 CUDA
3. Transformers：兼容性好，无需额外配置
"""

import os
from typing import Optional

# 默认系统提示词
DEFAULT_SYSTEM_PROMPT = """You are a prompt optimizer for image search. Convert Chinese queries into English descriptions for SigLIP model.
Rules:
1. Translate Chinese to English accurately
2. Expand into detailed visual descriptions (1-2 sentences)
3. Include visual attributes: colors, lighting, style, mood
4. Output ONLY the English prompt, nothing else

Examples:
- "日落海滩" -> "A beautiful sunset at the beach with warm orange and pink sky reflecting on calm ocean waves"
- "可爱的猫咪" -> "An adorable cute cat with fluffy fur and expressive eyes in a cozy home setting"
- "城市夜景" -> "Urban cityscape at night with illuminated skyscrapers and city lights" """


class PromptOptimizerService:
    """提示词优化服务 - 单例模式，支持多种推理后端"""

    _instance: Optional["PromptOptimizerService"] = None
    _initialized: bool = False

    def __new__(cls) -> "PromptOptimizerService":
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        if self._initialized:
            return

        self.model = None
        self.tokenizer = None
        self.sampling_params = None
        self.device = None
        self.backend = None  # "llama_cpp", "vllm", or "transformers"
        self._initialized = True

    def initialize(
            self,
            device: str,
            backend: str= "llama_cpp",
            gguf_model_path: Optional[str] = None,
            pytoch_model_path: Optional[str] = None,
    ) -> None:
        """
        初始化 Qwen3-0.6B 模型
        """
        if self.model is not None:
            return

        if backend == "llama_cpp":
            self._init_llama_cpp(gguf_model_path)
            return
        elif backend == "transformers":
            self._init_transformers(pytoch_model_path, device)
            return

    def _init_llama_cpp(self, gguf_model_path: Optional[str]) -> None:
        """使用 llama-cpp-python 初始化（CPU 优化）"""
        from llama_cpp import Llama

        # 如果未指定或文件不存在，自动下载
        if not gguf_model_path or not os.path.exists(gguf_model_path):
            gguf_model_path = self._download_gguf_model()

        print(f"[Prompt Optimizer] Loading GGUF model from {gguf_model_path}...")

        # CPU 线程数
        n_threads = int(os.environ.get("LLAMA_CPP_THREADS", os.cpu_count() or 4))

        self.model = Llama(
            model_path=gguf_model_path,
            n_ctx=1024,  # 上下文长度
            n_threads=n_threads,
            n_gpu_layers=0,  # CPU only
            verbose=False,
        )

        self.backend = "llama_cpp"
        self.device = "cpu"
        print(f"[Prompt Optimizer] llama.cpp backend initialized with {n_threads} threads!")

    def _download_gguf_model(self) -> str:
        """下载 GGUF 格式模型"""
        from huggingface_hub import hf_hub_download

        print("[Prompt Optimizer] Downloading Qwen3-0.6B GGUF model...")

        model_path = hf_hub_download(
            repo_id="Qwen/Qwen3-0.6B-GGUF",
            filename="Qwen3-0.6B-Q8_0.gguf",
        )

        return model_path

    def _init_transformers(self, pytoch_model_path: str, device: Optional[str]) -> None:
        """使用 Transformers 初始化"""
        from transformers import AutoModelForCausalLM, AutoTokenizer
        self.device = device
        print(f"[Prompt Optimizer] Loading {pytoch_model_path} with Transformers on {device}...")

        self.tokenizer = AutoTokenizer.from_pretrained(
            pytoch_model_path,
            trust_remote_code=True
        )
        self.model = AutoModelForCausalLM.from_pretrained(
            pytoch_model_path,
            torch_dtype="auto",
            device_map="auto" if device == "cuda" else None,
            trust_remote_code=True
        )

        if device != "cuda":
            self.model = self.model.to(device)

        self.model.eval()
        self.backend = "transformers"
        print("[Prompt Optimizer] Transformers backend initialized!")

    @property
    def is_loaded(self) -> bool:
        """检查模型是否已加载"""
        return self.model is not None

    def optimize_prompt(self, query: str, system_prompt: str) -> str:
        """优化搜索提示词"""
        if not self.is_loaded:
            raise RuntimeError("Model not loaded. Call initialize() first.")

        if system_prompt == '':
            system_prompt = None
        else:
            print('[Prompt Optimizer] use system prompt ', system_prompt.replace("\n", ""))

        if self.backend == "llama_cpp":
            return self._optimize_llama_cpp(query, system_prompt)
        else:
            return self._optimize_transformers(query, system_prompt)

    def _optimize_llama_cpp(self, query: str, systemPrompt: str) -> str:
        """使用 llama.cpp 推理"""
        # 构建 ChatML 格式提示词，添加 /no_think 禁用思考模式
        prompt = f"<|im_start|>system\n{DEFAULT_SYSTEM_PROMPT if (systemPrompt is None) else systemPrompt} /no_think<|im_end|>\n<|im_start|>user\n{query}<|im_end|>\n<|im_start|>assistant\n"

        output = self.model(
            prompt,
            max_tokens=128,  # 增加 token 以防思考模式
            temperature=0,  # 贪婪解码
            stop=["<|im_end|>", "<|endoftext|>"],
            echo=False,
        )

        response = output["choices"][0]["text"]
        return self._clean_response(response)

    def _optimize_transformers(self, query: str, systemPrompt: str) -> str:
        """使用 Transformers 推理"""
        import torch

        messages = [
            {"role": "system", "content": DEFAULT_SYSTEM_PROMPT if (systemPrompt is None) else systemPrompt},
            {"role": "user", "content": query}
        ]

        text = self.tokenizer.apply_chat_template(
            messages,
            tokenize=False,
            add_generation_prompt=True,
            enable_thinking=False
        )

        model_inputs = self.tokenizer([text], return_tensors="pt").to(self.device)

        with torch.inference_mode():
            generated_ids = self.model.generate(
                **model_inputs,
                max_new_tokens=64,
                do_sample=False,  # 贪婪解码
                pad_token_id=self.tokenizer.eos_token_id
            )

        output_ids = generated_ids[0][len(model_inputs.input_ids[0]):].tolist()
        response = self.tokenizer.decode(output_ids, skip_special_tokens=True)

        return self._clean_response(response)

    @staticmethod
    def _clean_response(response: str) -> str:
        """清理模型响应"""
        # 移除思考标签及其内容
        if "<think>" in response:
            # 提取 </think> 之后的内容
            if "</think>" in response:
                response = response.split("</think>")[-1]
            else:
                # 如果只有 <think> 没有 </think>，移除 <think> 之后的所有内容
                response = response.split("<think>")[0]

        response = response.strip().strip('"').strip("'")

        for prefix in ["Output:", "Optimized:", "Result:", "English:"]:
            if response.startswith(prefix):
                response = response[len(prefix):].strip()

        # 只取第一行
        response = response.split("\n")[0].strip()

        return response


# 全局服务实例
prompt_optimizer_service = PromptOptimizerService()
