"""
路由模块：定义API路由和处理函数
"""

import json
import sys
import time
import uuid
from datetime import datetime, timezone
from typing import Any, Dict, List, Optional, Union

import openai
import uvicorn
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse, StreamingResponse
from pydantic import ValidationError

from config.settings import settings
from models.anthropic import (MessagesRequest, MessagesResponse,
                             TokenCountRequest, TokenCountResponse)
from models.errors import (AnthropicErrorDetail, AnthropicErrorResponse,
                          AnthropicErrorType, ProviderErrorMetadata,
                          STATUS_CODE_ERROR_MAP)
from services.conversion import (convert_anthropic_to_openai_messages,
                                convert_anthropic_tool_choice_to_openai,
                                convert_anthropic_tools_to_openai,
                                convert_openai_to_anthropic_response,
                                extract_provider_error_details,
                                get_anthropic_error_details_from_exc,
                                select_target_model)
from services.streaming import handle_anthropic_streaming_response_from_openai_stream
from services.token_counting import count_tokens_for_anthropic_request
from utils.logging import (LogEvent, LogRecord, critical, debug, error,
                          info, warning)


# 创建OpenAI客户端
try:
    openai_client = openai.AsyncClient(
        api_key=settings.openai_api_key,
        base_url=settings.base_url,
        default_headers={
            "HTTP-Referer": settings.referrer_url,
            "X-Title": settings.app_name,
        },
        timeout=180.0,
    )
except Exception as e:
    critical(
        LogRecord(
            event="openai_client_init_failed",
            message="Failed to initialize OpenAI client",
        ),
        exc=e,
    )
    sys.exit(1)


def build_anthropic_error_response(
    error_type: AnthropicErrorType,
    message: str,
    status_code: int,
    provider_details: Optional[ProviderErrorMetadata] = None,
) -> JSONResponse:
    """
    创建带有Anthropic格式错误的JSONResponse。
    
    Args:
        error_type: Anthropic错误类型
        message: 错误消息
        status_code: HTTP状态码
        provider_details: 提供者错误元数据
        
    Returns:
        包含错误信息的JSONResponse
    """
    err_detail = AnthropicErrorDetail(type=error_type, message=message)
    if provider_details:
        err_detail.provider = provider_details.provider_name
        if provider_details.raw_error:
            if isinstance(provider_details.raw_error, dict):
                prov_err_obj = provider_details.raw_error.get("error")
                if isinstance(prov_err_obj, dict):
                    err_detail.provider_message = prov_err_obj.get("message")
                    err_detail.provider_code = prov_err_obj.get("code")
                elif isinstance(provider_details.raw_error.get("message"), str):
                    err_detail.provider_message = provider_details.raw_error.get(
                        "message"
                    )
                    err_detail.provider_code = provider_details.raw_error.get("code")

    error_resp_model = AnthropicErrorResponse(error=err_detail)
    return JSONResponse(
        status_code=status_code, content=error_resp_model.model_dump(exclude_unset=True)
    )


async def log_and_return_error_response(
    request: Request,
    status_code: int,
    anthropic_error_type: AnthropicErrorType,
    error_message: str,
    provider_details: Optional[ProviderErrorMetadata] = None,
    caught_exception: Optional[Exception] = None,
) -> JSONResponse:
    """
    记录错误并返回格式化的错误响应。
    
    Args:
        request: FastAPI请求对象
        status_code: HTTP状态码
        anthropic_error_type: Anthropic错误类型
        error_message: 错误消息
        provider_details: 提供者错误元数据
        caught_exception: 捕获的异常
        
    Returns:
        包含错误信息的JSONResponse
    """
    request_id = getattr(request.state, "request_id", "unknown")
    start_time_mono = getattr(request.state, "start_time_monotonic", time.monotonic())
    duration_ms = (time.monotonic() - start_time_mono) * 1000

    log_data = {
        "status_code": status_code,
        "duration_ms": duration_ms,
        "error_type": anthropic_error_type.value,
        "client_ip": request.client.host if request.client else "unknown",
    }
    if provider_details:
        log_data["provider_name"] = provider_details.provider_name
        log_data["provider_raw_error"] = provider_details.raw_error

    error(
        LogRecord(
            event=LogEvent.REQUEST_FAILURE.value,
            message=f"Request failed: {error_message}",
            request_id=request_id,
            data=log_data,
        ),
        exc=caught_exception,
    )
    return build_anthropic_error_response(
        anthropic_error_type, error_message, status_code, provider_details
    )


def create_routes(app: FastAPI) -> None:
    """
    创建API路由。
    
    Args:
        app: FastAPI应用实例
    """
    
    @app.post("/v1/messages", response_model=None, tags=["API"], status_code=200)
    async def create_message_proxy(
        request: Request,
    ) -> Union[JSONResponse, StreamingResponse]:
        """
        Anthropic消息完成的主要端点，代理到OpenAI兼容的API。
        处理请求/响应转换、流式传输和动态模型选择。
        支持提示词缓存功能。
        """
        request_id = str(uuid.uuid4())
        request.state.request_id = request_id
        request.state.start_time_monotonic = time.monotonic()

        try:
            raw_body = await request.json()
            debug(
                LogRecord(
                    LogEvent.ANTHROPIC_REQUEST.value,
                    "Received Anthropic request body",
                    request_id,
                    {"body": raw_body},
                )
            )

            anthropic_request = MessagesRequest.model_validate(
                raw_body, context={"request_id": request_id}
            )
        except json.JSONDecodeError as e:
            return await log_and_return_error_response(
                request,
                400,
                AnthropicErrorType.INVALID_REQUEST,
                "Invalid JSON body.",
                caught_exception=e,
            )
        except ValidationError as e:
            return await log_and_return_error_response(
                request,
                422,
                AnthropicErrorType.INVALID_REQUEST,
                f"Invalid request body: {e.errors()}",
                caught_exception=e,
            )

        is_stream = anthropic_request.stream or False
        target_model_name = select_target_model(anthropic_request.model, request_id, settings)

        estimated_input_tokens = count_tokens_for_anthropic_request(
            messages=anthropic_request.messages,
            system=anthropic_request.system,
            model_name=anthropic_request.model,
            tools=anthropic_request.tools,
            request_id=request_id,
        )

        info(
            LogRecord(
                event=LogEvent.REQUEST_START.value,
                message="Processing new message request",
                request_id=request_id,
                data={
                    "client_model": anthropic_request.model,
                    "target_model": target_model_name,
                    "stream": is_stream,
                    "estimated_input_tokens": estimated_input_tokens,
                    "client_ip": request.client.host if request.client else "unknown",
                    "user_agent": request.headers.get("user-agent", "unknown"),
                },
            )
        )

        try:
            openai_messages = convert_anthropic_to_openai_messages(
                anthropic_request.messages,
                target_model_name,
                anthropic_request.system,
                request_id=request_id,
            )
            openai_tools = convert_anthropic_tools_to_openai(
                anthropic_request.tools, target_model_name
            )
            openai_tool_choice = convert_anthropic_tool_choice_to_openai(
                anthropic_request.tool_choice, request_id
            )
        except Exception as e:
            return await log_and_return_error_response(
                request,
                500,
                AnthropicErrorType.API_ERROR,
                "Error during request conversion.",
                caught_exception=e,
            )

        openai_params: Dict[str, Any] = {
            "model": target_model_name,
            "messages": openai_messages,
            "max_tokens": anthropic_request.max_tokens,
            "stream": is_stream,
        }
        if anthropic_request.temperature is not None:
            openai_params["temperature"] = anthropic_request.temperature
        if anthropic_request.top_p is not None:
            openai_params["top_p"] = anthropic_request.top_p
        if anthropic_request.stop_sequences:
            openai_params["stop"] = anthropic_request.stop_sequences
        if openai_tools:
            openai_params["tools"] = openai_tools
        if openai_tool_choice:
            openai_params["tool_choice"] = openai_tool_choice
        if anthropic_request.metadata and anthropic_request.metadata.get("user_id"):
            openai_params["user"] = str(anthropic_request.metadata.get("user_id"))

        debug(
            LogRecord(
                LogEvent.OPENAI_REQUEST.value,
                "Prepared OpenAI request parameters",
                request_id,
                {"params": openai_params},
            )
        )

        try:
            # 记录转换日志，确保保留原始请求和转换后的请求
            log_entry = {
                "request_id": request_id,
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "anthropic_request": raw_body,
                "openai_request": openai_params,
            }
            with open("conversion_log.jsonl", "a", encoding="utf-8") as f:
                f.write(json.dumps(log_entry, ensure_ascii=False, indent=2))
                f.write("\n")
        except Exception as e:
            warning(
                LogRecord(
                    event="conversion_log_failure",
                    message="Failed to write to conversion_log.jsonl",
                    request_id=request_id,
                ),
                exc=e,
            )

        try:
            if is_stream:
                debug(
                    LogRecord(
                        LogEvent.STREAMING_REQUEST.value,
                        "Initiating streaming request to OpenAI-compatible API",
                        request_id,
                    )
                )
                openai_stream_response = await openai_client.chat.completions.create(
                    **openai_params
                )
                return StreamingResponse(
                    handle_anthropic_streaming_response_from_openai_stream(
                        openai_stream_response,
                        anthropic_request.model,
                        estimated_input_tokens,
                        request_id,
                        request.state.start_time_monotonic,
                    ),
                    media_type="text/event-stream",
                )
            else:
                debug(
                    LogRecord(
                        LogEvent.OPENAI_REQUEST.value,
                        "Sending non-streaming request to OpenAI-compatible API",
                        request_id,
                    )
                )
                openai_response_obj = await openai_client.chat.completions.create(
                    **openai_params
                )

                debug(
                    LogRecord(
                        LogEvent.OPENAI_RESPONSE.value,
                        "Received OpenAI response",
                        request_id,
                        {"response": openai_response_obj.model_dump()},
                    )
                )

                anthropic_response_obj = convert_openai_to_anthropic_response(
                    openai_response_obj, anthropic_request.model, request_id=request_id
                )
                duration_ms = (time.monotonic() - request.state.start_time_monotonic) * 1000
                info(
                    LogRecord(
                        event=LogEvent.REQUEST_COMPLETED.value,
                        message="Non-streaming request completed successfully",
                        request_id=request_id,
                        data={
                            "status_code": 200,
                            "duration_ms": duration_ms,
                            "input_tokens": anthropic_response_obj.usage.input_tokens,
                            "output_tokens": anthropic_response_obj.usage.output_tokens,
                            "stop_reason": anthropic_response_obj.stop_reason,
                        },
                    )
                )
                debug(
                    LogRecord(
                        LogEvent.ANTHROPIC_RESPONSE.value,
                        "Prepared Anthropic response",
                        request_id,
                        {"response": anthropic_response_obj.model_dump(exclude_unset=True)},
                    )
                )
                return JSONResponse(
                    content=anthropic_response_obj.model_dump(exclude_unset=True)
                )

        except openai.APIError as e:
            err_type, err_msg, err_status, prov_details = (
                get_anthropic_error_details_from_exc(e)
            )
            return await log_and_return_error_response(
                request, err_status, err_type, err_msg, prov_details, e
            )
        except Exception as e:
            return await log_and_return_error_response(
                request,
                500,
                AnthropicErrorType.API_ERROR,
                "An unexpected error occurred while processing the request.",
                caught_exception=e,
            )

    @app.post(
        "/v1/messages/count_tokens", response_model=TokenCountResponse, tags=["Utility"]
    )
    async def count_tokens_endpoint(request: Request) -> TokenCountResponse:
        """估计给定Anthropic消息和系统提示的令牌数量，支持缓存控制。"""
        request_id = str(uuid.uuid4())
        request.state.request_id = request_id
        start_time_mono = time.monotonic()

        try:
            body = await request.json()
            count_request = TokenCountRequest.model_validate(body)
        except json.JSONDecodeError as e:
            raise HTTPException(status_code=400, detail="Invalid JSON body.") from e
        except ValidationError as e:
            raise HTTPException(
                status_code=422, detail=f"Invalid request body: {e.errors()}"
            ) from e

        token_count = count_tokens_for_anthropic_request(
            messages=count_request.messages,
            system=count_request.system,
            model_name=count_request.model,
            tools=count_request.tools,
            request_id=request_id,
        )
        duration_ms = (time.monotonic() - start_time_mono) * 1000
        info(
            LogRecord(
                event=LogEvent.TOKEN_COUNT.value,
                message=f"Counted {token_count} tokens",
                request_id=request_id,
                data={
                    "duration_ms": duration_ms,
                    "token_count": token_count,
                    "model": count_request.model,
                },
            )
        )
        return TokenCountResponse(input_tokens=token_count)

    @app.get("/", include_in_schema=False, tags=["Health"])
    async def root_health_check() -> JSONResponse:
        """基本健康检查和信息端点。"""
        debug(
            LogRecord(
                event=LogEvent.HEALTH_CHECK.value, message="Root health check accessed"
            )
        )
        return JSONResponse(
            {
                "proxy_name": settings.app_name,
                "version": settings.app_version,
                "status": "ok",
                "timestamp": datetime.now(timezone.utc).isoformat(),
            }
        )
