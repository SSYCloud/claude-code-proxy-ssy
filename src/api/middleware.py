"""
中间件模块：提供FastAPI中间件功能
"""

import time
from typing import Awaitable, Callable

from fastapi import Request, Response

from utils.logging import LogRecord


async def logging_middleware(
    request: Request, call_next: Callable[[Request], Awaitable[Response]]
) -> Response:
    """
    日志记录中间件，记录请求处理时间并添加请求ID到响应头。
    
    Args:
        request: FastAPI请求对象
        call_next: 下一个中间件或路由处理函数
        
    Returns:
        FastAPI响应对象
    """
    # 确保请求状态中有请求ID和开始时间
    if not hasattr(request.state, "request_id"):
        request.state.request_id = "unknown"
    if not hasattr(request.state, "start_time_monotonic"):
        request.state.start_time_monotonic = time.monotonic()

    # 处理请求
    response = await call_next(request)

    # 添加响应头
    response.headers["X-Request-ID"] = request.state.request_id
    duration_ms = (time.monotonic() - request.state.start_time_monotonic) * 1000
    response.headers["X-Response-Time-ms"] = str(duration_ms)

    return response
