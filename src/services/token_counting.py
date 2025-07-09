"""
令牌计数服务模块：提供计算Anthropic请求令牌数量的功能
"""

import json
from typing import Dict, List, Optional, Union

import tiktoken

from models.anthropic import (ContentBlockImage, ContentBlockText,
                                  ContentBlockToolResult, ContentBlockToolUse,
                                  Message, SystemContent, Tool)
from utils.logging import LogEvent, LogRecord, debug, warning


# 令牌编码器缓存
_token_encoder_cache: Dict[str, tiktoken.Encoding] = {}


def get_token_encoder(
    model_name: str = "gpt-4", request_id: Optional[str] = None
) -> tiktoken.Encoding:
    """
    获取tiktoken编码器，并缓存以提高性能。
    
    Args:
        model_name: 模型名称
        request_id: 请求ID，用于日志记录
        
    Returns:
        tiktoken编码器实例
    """
    cache_key = "gpt-4"
    if cache_key not in _token_encoder_cache:
        try:
            _token_encoder_cache[cache_key] = tiktoken.encoding_for_model(cache_key)
        except Exception:
            try:
                _token_encoder_cache[cache_key] = tiktoken.get_encoding("cl100k_base")
                warning(
                    LogRecord(
                        event=LogEvent.TOKEN_ENCODER_LOAD_FAILED.value,
                        message=f"Could not load tiktoken encoder for '{cache_key}', using 'cl100k_base'. Token counts may be approximate.",
                        request_id=request_id,
                        data={"model_tried": cache_key},
                    )
                )
            except Exception as e_cl:
                warning(
                    LogRecord(
                        event=LogEvent.TOKEN_ENCODER_LOAD_FAILED.value,
                        message="Failed to load any tiktoken encoder (gpt-4, cl100k_base). Token counting will be inaccurate.",
                        request_id=request_id,
                    ),
                    exc=e_cl,
                )

                class DummyEncoder:
                    def encode(self, text: str) -> List[int]:
                        return list(range(len(text)))

                _token_encoder_cache[cache_key] = DummyEncoder()
    return _token_encoder_cache[cache_key]


def count_tokens_for_anthropic_request(
    messages: List[Message],
    system: Optional[Union[str, List[SystemContent]]],
    model_name: str,
    tools: Optional[List[Tool]] = None,
    request_id: Optional[str] = None,
) -> int:
    """
    计算Anthropic请求的令牌数量，包括支持缓存控制。
    
    Args:
        messages: 消息列表
        system: 系统提示
        model_name: 模型名称
        tools: 工具列表
        request_id: 请求ID，用于日志记录
        
    Returns:
        估计的令牌数量
    """
    enc = get_token_encoder(model_name, request_id)
    total_tokens = 0

    # 计算系统提示的令牌数
    if isinstance(system, str):
        total_tokens += len(enc.encode(system))
    elif isinstance(system, list):
        for block in system:
            if isinstance(block, SystemContent) and block.type == "text":
                total_tokens += len(enc.encode(block.text))
                # 为缓存控制添加令牌（如果存在）
                if block.cache_control:
                    # 缓存控制结构的近似令牌数
                    total_tokens += 5

    # 计算消息的令牌数
    for msg in messages:
        total_tokens += 4  # 消息格式开销
        if msg.role:
            total_tokens += len(enc.encode(msg.role))

        if isinstance(msg.content, str):
            total_tokens += len(enc.encode(msg.content))
        elif isinstance(msg.content, list):
            for block in msg.content:
                if isinstance(block, ContentBlockText):
                    total_tokens += len(enc.encode(block.text))
                    # 为缓存控制添加令牌（如果存在）
                    if block.cache_control:
                        total_tokens += 5
                elif isinstance(block, ContentBlockImage):
                    # 图像通常消耗固定数量的令牌
                    total_tokens += 768
                    # 为缓存控制添加令牌（如果存在）
                    if block.cache_control:
                        total_tokens += 5
                elif isinstance(block, ContentBlockToolUse):
                    total_tokens += len(enc.encode(block.name))
                    # 为缓存控制添加令牌（如果存在）
                    if block.cache_control:
                        total_tokens += 5
                    try:
                        input_str = json.dumps(block.input)
                        total_tokens += len(enc.encode(input_str))
                    except Exception:
                        warning(
                            LogRecord(
                                event=LogEvent.TOOL_INPUT_SERIALIZATION_FAILURE.value,
                                message="Failed to serialize tool input for token counting.",
                                data={"tool_name": block.name},
                                request_id=request_id,
                            )
                        )
                elif isinstance(block, ContentBlockToolResult):
                    try:
                        content_str = ""
                        if isinstance(block.content, str):
                            content_str = block.content
                        elif isinstance(block.content, list):
                            for item in block.content:
                                if (
                                    isinstance(item, dict)
                                    and item.get("type") == "text"
                                ):
                                    content_str += item.get("text", "")
                                else:
                                    content_str += json.dumps(item)
                        else:
                            content_str = json.dumps(block.content)
                        total_tokens += len(enc.encode(content_str))
                        # 为缓存控制添加令牌（如果存在）
                        if block.cache_control:
                            total_tokens += 5
                    except Exception:
                        warning(
                            LogRecord(
                                event=LogEvent.TOOL_RESULT_SERIALIZATION_FAILURE.value,
                                message="Failed to serialize tool result for token counting.",
                                request_id=request_id,
                            )
                        )

    # 计算工具的令牌数
    if tools:
        total_tokens += 2  # 工具列表开销
        for tool in tools:
            total_tokens += len(enc.encode(tool.name))
            if tool.description:
                total_tokens += len(enc.encode(tool.description))
            try:
                schema_str = json.dumps(tool.input_schema)
                total_tokens += len(enc.encode(schema_str))
                # 为缓存控制添加令牌（如果存在）
                if tool.cache_control:
                    total_tokens += 5
            except Exception:
                warning(
                    LogRecord(
                        event=LogEvent.TOOL_INPUT_SERIALIZATION_FAILURE.value,
                        message="Failed to serialize tool schema for token counting.",
                        data={"tool_name": tool.name},
                        request_id=request_id,
                    )
                )
    
    debug(
        LogRecord(
            event=LogEvent.TOKEN_COUNT.value,
            message=f"Estimated {total_tokens} input tokens for model {model_name}",
            data={"model": model_name, "token_count": total_tokens},
            request_id=request_id,
        )
    )
    return total_tokens
