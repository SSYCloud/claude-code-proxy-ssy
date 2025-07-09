"""
主入口模块：应用程序的入口点，负责初始化和启动FastAPI应用
"""

import sys

import fastapi
import uvicorn
from fastapi import Request
from fastapi.responses import JSONResponse

from api.middleware import logging_middleware
from api.routes import create_routes
from config.settings import settings
from models.errors import AnthropicErrorType
from utils.logging import (LogEvent, LogRecord, critical, error, print_startup_banner)


def create_app() -> fastapi.FastAPI:
    """
    创建并配置FastAPI应用实例
    
    Returns:
        配置好的FastAPI应用实例
    """
    app = fastapi.FastAPI(
        title=settings.app_name,
        description="Routes Anthropic API requests to an OpenAI-compatible API, selecting models dynamically.",
        version=settings.app_version,
        docs_url=None,
        redoc_url=None,
    )

    # 添加中间件
    app.middleware("http")(logging_middleware)

    # 创建路由
    create_routes(app)

    # 添加异常处理器
    @app.exception_handler(Exception)
    async def generic_exception_handler(request: Request, exc: Exception):
        """通用异常处理器"""
        return await log_and_return_error_response(
            request,
            500,
            AnthropicErrorType.API_ERROR,
            "An unexpected internal server error occurred.",
            caught_exception=exc,
        )

    @app.exception_handler(fastapi.exceptions.RequestValidationError)
    async def validation_exception_handler(request: Request, exc: fastapi.exceptions.RequestValidationError):
        """请求验证异常处理器"""
        return await log_and_return_error_response(
            request,
            422,
            AnthropicErrorType.INVALID_REQUEST,
            f"Validation error: {exc.errors()}",
            caught_exception=exc,
        )

    return app


async def log_and_return_error_response(
    request: Request,
    status_code: int,
    error_type: AnthropicErrorType,
    error_message: str,
    caught_exception: Exception = None,
):
    """
    记录错误并返回标准错误响应
    
    Args:
        request: FastAPI请求对象
        status_code: HTTP状态码
        error_type: Anthropic错误类型
        error_message: 错误消息
        caught_exception: 捕获的异常
        
    Returns:
        包含错误信息的JSONResponse
    """
    request_id = getattr(request.state, "request_id", "unknown")
    
    error(
        LogRecord(
            event=LogEvent.REQUEST_FAILURE.value,
            message=f"Request failed: {error_message}",
            request_id=request_id,
            data={"status_code": status_code, "error_type": error_type.value},
        ),
        exc=caught_exception,
    )
    
    return JSONResponse(
        status_code=status_code,
        content={
            "type": "error",
            "error": {
                "type": error_type.value,
                "message": error_message,
            },
        },
    )


app = create_app()


if __name__ == "__main__":
    try:
        print_startup_banner()
        uvicorn.run(
            "main:app",
            host=settings.host,
            port=settings.port,
            reload=settings.reload,
            log_config=None,
            access_log=False,
        )
    except Exception as e:
        critical(
            LogRecord(
                event="startup_failed",
                message="Failed to start application",
            ),
            exc=e,
        )
        sys.exit(1)
