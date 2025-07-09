"""
错误模型模块：定义API错误响应的数据模型
"""

import enum
from typing import Dict, Any, Optional, Union

from pydantic import BaseModel


class AnthropicErrorType(str, enum.Enum):
    """Anthropic API错误类型枚举"""
    INVALID_REQUEST = "invalid_request_error"
    AUTHENTICATION = "authentication_error"
    PERMISSION = "permission_error"
    NOT_FOUND = "not_found_error"
    RATE_LIMIT = "rate_limit_error"
    API_ERROR = "api_error"
    OVERLOADED = "overloaded_error"
    REQUEST_TOO_LARGE = "request_too_large_error"


class AnthropicErrorDetail(BaseModel):
    """Anthropic错误详情模型"""
    type: AnthropicErrorType
    message: str
    provider: Optional[str] = None
    provider_message: Optional[str] = None
    provider_code: Optional[Union[str, int]] = None


class AnthropicErrorResponse(BaseModel):
    """Anthropic错误响应模型"""
    type: str = "error"
    error: AnthropicErrorDetail


class ProviderErrorMetadata(BaseModel):
    """提供者错误元数据模型"""
    provider_name: str
    raw_error: Optional[Dict[str, Any]] = None


# HTTP状态码到Anthropic错误类型的映射
STATUS_CODE_ERROR_MAP: Dict[int, AnthropicErrorType] = {
    400: AnthropicErrorType.INVALID_REQUEST,
    401: AnthropicErrorType.AUTHENTICATION,
    403: AnthropicErrorType.PERMISSION,
    404: AnthropicErrorType.NOT_FOUND,
    413: AnthropicErrorType.REQUEST_TOO_LARGE,
    422: AnthropicErrorType.INVALID_REQUEST,
    429: AnthropicErrorType.RATE_LIMIT,
    500: AnthropicErrorType.API_ERROR,
    502: AnthropicErrorType.API_ERROR,
    503: AnthropicErrorType.OVERLOADED,
    504: AnthropicErrorType.API_ERROR,
}
