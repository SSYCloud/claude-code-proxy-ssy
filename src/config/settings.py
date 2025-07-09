"""
配置模块：从环境变量加载应用程序设置
"""

import os
from typing import Optional
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """从环境变量加载的应用程序设置。"""

    model_config = SettingsConfigDict(
        env_file="../.env", 
        env_file_encoding="utf-8",
        extra="ignore",
        env_prefix="",  # 不使用前缀
        env_ignore_empty=False,  # 不忽略空值
        case_sensitive=False,  # 不区分大小写
    )

    openai_api_key: str
    big_model_name: str
    small_model_name: str
    #base_url: str = "https://openrouter.ai/api/v1"
    base_url: str = "https://router.shengsuanyun.com/api/v1"

    referrer_url: str = "http://localhost:8080/claude_proxy"

    app_name: str = "AnthropicProxy"
    app_version: str = "0.2.0"
    log_level: str = "INFO"
    log_file_path: Optional[str] = "log.jsonl"
    host: str = "127.0.0.1"
    port: int = 8080
    reload: bool = True
    open_claude_cache: bool = False

# 创建全局设置实例
settings = Settings()
