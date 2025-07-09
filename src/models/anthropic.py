"""
Anthropic模型模块：定义Anthropic API请求和响应的数据模型
"""

from typing import Any, Dict, List, Literal, Optional, Union
from pydantic import BaseModel, Field, field_validator

from utils.logging import LogEvent, LogRecord, warning


class CacheControl(BaseModel):
    """缓存控制模型，用于提示词缓存功能。"""
    type: Literal["ephemeral"]


class ContentBlockText(BaseModel):
    """文本内容块模型"""
    type: Literal["text"]
    text: str
    cache_control: Optional[CacheControl] = None


class ContentBlockImageSource(BaseModel):
    """图像源模型"""
    type: str
    media_type: str
    data: str


class ContentBlockImage(BaseModel):
    """图像内容块模型"""
    type: Literal["image"]
    source: ContentBlockImageSource
    cache_control: Optional[CacheControl] = None


class ContentBlockToolUse(BaseModel):
    """工具使用内容块模型"""
    type: Literal["tool_use"]
    id: str
    name: str
    input: Dict[str, Any]
    cache_control: Optional[CacheControl] = None


class ContentBlockToolResult(BaseModel):
    """工具结果内容块模型"""
    type: Literal["tool_result"]
    tool_use_id: str
    content: Union[str, List[Dict[str, Any]], List[Any]]
    is_error: Optional[bool] = None
    cache_control: Optional[CacheControl] = None


# 内容块联合类型
ContentBlock = Union[
    ContentBlockText, ContentBlockImage, ContentBlockToolUse, ContentBlockToolResult
]


class SystemContent(BaseModel):
    """系统内容模型"""
    type: Literal["text"]
    text: str
    cache_control: Optional[CacheControl] = None


class Message(BaseModel):
    """消息模型"""
    role: Literal["user", "assistant"]
    content: Union[str, List[ContentBlock]]


class Tool(BaseModel):
    """工具定义模型"""
    name: str
    description: Optional[str] = None
    input_schema: Dict[str, Any] = Field(..., alias="input_schema")
    cache_control: Optional[CacheControl] = None


class ToolChoice(BaseModel):
    """工具选择模型"""
    type: Literal["auto", "any", "tool"]
    name: Optional[str] = None


class MessagesRequest(BaseModel):
    """消息请求模型"""
    model: str
    max_tokens: int
    messages: List[Message]
    system: Optional[Union[str, List[SystemContent]]] = None
    stop_sequences: Optional[List[str]] = None
    stream: Optional[bool] = False
    temperature: Optional[float] = None
    top_p: Optional[float] = None
    top_k: Optional[int] = None
    metadata: Optional[Dict[str, Any]] = None
    tools: Optional[List[Tool]] = None
    tool_choice: Optional[ToolChoice] = None

    @field_validator("top_k")
    def check_top_k(cls, v: Optional[int]) -> Optional[int]:
        """验证top_k参数，OpenAI不直接支持此参数"""
        if v is not None:
            # 在验证器中无法访问请求上下文，所以这里不设置request_id
            warning(
                LogRecord(
                    event=LogEvent.PARAMETER_UNSUPPORTED.value,
                    message="Parameter 'top_k' provided by client but is not directly supported by the OpenAI Chat Completions API and will be ignored.",
                    request_id=None,
                    data={"parameter": "top_k", "value": v},
                )
            )
        return v


class TokenCountRequest(BaseModel):
    """令牌计数请求模型"""
    model: str
    messages: List[Message]
    system: Optional[Union[str, List[SystemContent]]] = None
    tools: Optional[List[Tool]] = None


class TokenCountResponse(BaseModel):
    """令牌计数响应模型"""
    input_tokens: int


class Usage(BaseModel):
    """使用情况模型"""
    input_tokens: int
    output_tokens: int


class MessagesResponse(BaseModel):
    """消息响应模型"""
    id: str
    type: Literal["message"] = "message"
    role: Literal["assistant"] = "assistant"
    model: str
    content: List[ContentBlock]
    stop_reason: Optional[
        Literal["end_turn", "max_tokens", "stop_sequence", "tool_use", "error"]
    ] = None
    stop_sequence: Optional[str] = None
    usage: Usage
