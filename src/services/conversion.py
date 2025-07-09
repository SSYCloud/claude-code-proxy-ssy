"""
转换服务模块：提供Anthropic和OpenAI格式之间的转换功能
"""

import json
from typing import Any, Dict, List, Literal, Optional, Tuple, Union, cast

from openai.types.chat import ChatCompletionMessageParam, ChatCompletionToolParam

from models.anthropic import (ContentBlock, ContentBlockImage,
                                  ContentBlockText, ContentBlockToolResult,
                                  ContentBlockToolUse, Message, MessagesResponse,
                                  SystemContent, Tool, ToolChoice, Usage)
from models.errors import AnthropicErrorType, ProviderErrorMetadata
from config.settings import settings
from utils.logging import LogEvent, LogRecord, debug, error, warning


def extract_provider_error_details(
    error_details_dict: Optional[Dict[str, Any]],
) -> Optional[ProviderErrorMetadata]:
    """
    从错误详情字典中提取提供者错误元数据
    
    Args:
        error_details_dict: 错误详情字典
        
    Returns:
        提供者错误元数据，如果无法提取则返回None
    """
    if not isinstance(error_details_dict, dict):
        return None
    metadata = error_details_dict.get("metadata")
    if not isinstance(metadata, dict):
        return None
    provider_name = metadata.get("provider_name")
    raw_error_str = metadata.get("raw")

    if not provider_name or not isinstance(provider_name, str):
        return None

    parsed_raw_error: Optional[Dict[str, Any]] = None
    if isinstance(raw_error_str, str):
        try:
            parsed_raw_error = json.loads(raw_error_str)
        except json.JSONDecodeError:
            warning(
                LogRecord(
                    event=LogEvent.PROVIDER_ERROR_DETAILS.value,
                    message=f"Failed to parse raw provider error string for {provider_name}.",
                )
            )
            parsed_raw_error = {"raw_string_parse_failed": raw_error_str}
    elif isinstance(raw_error_str, dict):
        parsed_raw_error = raw_error_str

    return ProviderErrorMetadata(
        provider_name=provider_name, raw_error=parsed_raw_error
    )


def _serialize_tool_result_content_for_openai(
    anthropic_tool_result_content: Union[str, List[Dict[str, Any]], List[Any]],
    request_id: Optional[str],
    log_context: Dict,
) -> str:
    """
    将Anthropic工具结果内容（可能很复杂）序列化为单个字符串，
    以满足OpenAI对'tool'角色消息的'content'字段的要求。
    
    Args:
        anthropic_tool_result_content: Anthropic工具结果内容
        request_id: 请求ID，用于日志记录
        log_context: 日志上下文
        
    Returns:
        序列化后的字符串
    """
    if isinstance(anthropic_tool_result_content, str):
        return anthropic_tool_result_content

    if isinstance(anthropic_tool_result_content, list):
        processed_parts = []
        contains_non_text_block = False
        for item in anthropic_tool_result_content:
            if isinstance(item, dict) and item.get("type") == "text" and "text" in item:
                processed_parts.append(str(item["text"]))
            else:
                try:
                    processed_parts.append(json.dumps(item))
                    contains_non_text_block = True
                except TypeError:
                    processed_parts.append(
                        f"<unserializable_item type='{type(item).__name__}'>"
                    )
                    contains_non_text_block = True

        result_str = "\n".join(processed_parts)
        if contains_non_text_block:
            warning(
                LogRecord(
                    event=LogEvent.TOOL_RESULT_PROCESSING.value,
                    message="Tool result content list contained non-text or complex items; parts were JSON stringified.",
                    request_id=request_id,
                    data={**log_context, "result_str_preview": result_str[:100]},
                )
            )
        return result_str

    try:
        return json.dumps(anthropic_tool_result_content)
    except TypeError as e:
        warning(
            LogRecord(
                event=LogEvent.TOOL_RESULT_SERIALIZATION_FAILURE.value,
                message=f"Failed to serialize tool result content to JSON: {e}. Returning error JSON.",
                request_id=request_id,
                data=log_context,
            )
        )
        return json.dumps(
            {
                "error": "Serialization failed",
                "original_type": str(type(anthropic_tool_result_content)),
            }
        )


def convert_anthropic_to_openai_messages(
    anthropic_messages: List[Message],
    target_model_name: str,
    anthropic_system: Optional[Union[str, List[SystemContent]]] = None,
    request_id: Optional[str] = None,
) -> List[Dict[str, Any]]:
    """
    将Anthropic消息格式转换为OpenAI消息格式，支持缓存控制。
    
    Args:
        anthropic_messages: Anthropic格式的消息列表
        target_model_name: 目标模型名称
        anthropic_system: Anthropic系统提示
        request_id: 请求ID，用于日志记录
        
    Returns:
        OpenAI格式的消息列表
    """
    openai_messages: List[Dict[str, Any]] = []
    if settings.open_claude_cache and "claude" in target_model_name.lower():
        is_claude_model = True
    else:
        is_claude_model = False

    # 处理系统提示
    if anthropic_system:
        if isinstance(anthropic_system, str):
            openai_messages.append({"role": "system", "content": anthropic_system})
        elif isinstance(anthropic_system, list):
            if is_claude_model:
                # 对于Claude模型，将系统内容作为块列表传递
                system_content_blocks = []
                for block in anthropic_system:
                    if isinstance(block, SystemContent):
                        block_dict = block.model_dump(exclude_unset=True)
                        if not block.cache_control:
                            block_dict.pop("cache_control", None)
                        system_content_blocks.append(block_dict)
                if system_content_blocks:
                    openai_messages.append(
                        {"role": "system", "content": system_content_blocks}
                    )
            else:
                # 对于其他模型，连接文本块
                system_texts = []
                for block in anthropic_system:
                    if isinstance(block, SystemContent) and block.type == "text":
                        system_texts.append(block.text)

                if len(system_texts) < len(anthropic_system):
                    warning(
                        LogRecord(
                            event=LogEvent.SYSTEM_PROMPT_ADJUSTED.value,
                            message="Non-text content blocks in Anthropic system prompt were ignored for non-Claude model.",
                            request_id=request_id,
                        )
                    )
                system_text_content = "\n".join(system_texts)
                if system_text_content:
                    openai_messages.append(
                        {"role": "system", "content": system_text_content}
                    )

    # 处理消息
    for i, msg in enumerate(anthropic_messages):
        role = msg.role
        content = msg.content

        if isinstance(content, str):
            openai_messages.append({"role": role, "content": content})
            continue

        if isinstance(content, list):
            openai_parts_for_user_message = []
            assistant_tool_calls = []
            text_content_for_assistant = []

            if not content and role == "user":
                openai_messages.append({"role": "user", "content": ""})
                continue
            if not content and role == "assistant":
                openai_messages.append({"role": "assistant", "content": ""})
                continue

            for block_idx, block in enumerate(content):
                block_log_ctx = {
                    "anthropic_message_index": i,
                    "block_index": block_idx,
                    "block_type": block.type,
                }

                if isinstance(block, ContentBlockText):
                    if role == "user":
                        text_part = {"type": "text", "text": block.text}
                        if is_claude_model and block.cache_control:
                            text_part["cache_control"] = block.cache_control.model_dump()
                        openai_parts_for_user_message.append(text_part)
                    elif role == "assistant":
                        text_content_for_assistant.append(block.text)

                elif isinstance(block, ContentBlockImage) and role == "user":
                    if block.source.type == "base64":
                        image_part = {
                            "type": "image_url",
                            "image_url": {
                                "url": f"data:{block.source.media_type};base64,{block.source.data}"
                            },
                        }
                        if is_claude_model and block.cache_control:
                            image_part["cache_control"] = block.cache_control.model_dump()
                        openai_parts_for_user_message.append(image_part)
                    else:
                        warning(
                            LogRecord(
                                event=LogEvent.IMAGE_FORMAT_UNSUPPORTED.value,
                                message=f"Image block with source type '{block.source.type}' (expected 'base64') ignored in user message {i}.",
                                request_id=request_id,
                                data=block_log_ctx,
                            )
                        )

                elif isinstance(block, ContentBlockToolUse) and role == "assistant":
                    try:
                        args_str = json.dumps(block.input)
                    except Exception as e:
                        error(
                            LogRecord(
                                event=LogEvent.TOOL_INPUT_SERIALIZATION_FAILURE.value,
                                message=f"Failed to serialize tool input for tool '{block.name}'. Using empty JSON.",
                                request_id=request_id,
                                data={
                                    **block_log_ctx,
                                    "tool_id": block.id,
                                    "tool_name": block.name,
                                },
                            ),
                            exc=e,
                        )
                        args_str = "{}"

                    tool_call = {
                        "id": block.id,
                        "type": "function",
                        "function": {"name": block.name, "arguments": args_str},
                    }
                    if is_claude_model and block.cache_control:
                        tool_call["cache_control"] = block.cache_control.model_dump()
                    assistant_tool_calls.append(tool_call)

                elif isinstance(block, ContentBlockToolResult) and role == "user":
                    serialized_content = _serialize_tool_result_content_for_openai(
                        block.content, request_id, block_log_ctx
                    )
                    tool_result_message = {
                        "role": "tool",
                        "tool_call_id": block.tool_use_id,
                        "content": serialized_content,
                    }
                    if is_claude_model and block.cache_control:
                        tool_result_message[
                            "cache_control"
                        ] = block.cache_control.model_dump()
                    openai_messages.append(tool_result_message)

            if role == "user" and openai_parts_for_user_message:
                is_multimodal = any(
                    part["type"] == "image_url"
                    for part in openai_parts_for_user_message
                )
                if is_multimodal or len(openai_parts_for_user_message) > 1:
                    openai_messages.append(
                        {"role": "user", "content": openai_parts_for_user_message}
                    )
                elif (
                    len(openai_parts_for_user_message) == 1
                    and openai_parts_for_user_message[0]["type"] == "text"
                ):
                    openai_messages.append(
                        {
                            "role": "user",
                            "content": openai_parts_for_user_message[0]["text"],
                        }
                    )
                elif not openai_parts_for_user_message:
                    openai_messages.append({"role": "user", "content": ""})

            if role == "assistant":
                assistant_text = "\n".join(filter(None, text_content_for_assistant))
                if assistant_text:
                    openai_messages.append(
                        {"role": "assistant", "content": assistant_text}
                    )

                if assistant_tool_calls:
                    if (
                        openai_messages
                        and openai_messages[-1]["role"] == "assistant"
                        and openai_messages[-1].get("content")
                    ):
                        openai_messages.append(
                            {
                                "role": "assistant",
                                "content": None,
                                "tool_calls": assistant_tool_calls,
                            }
                        )

                    elif (
                        openai_messages
                        and openai_messages[-1]["role"] == "assistant"
                        and not openai_messages[-1].get("tool_calls")
                    ):
                        openai_messages[-1]["tool_calls"] = assistant_tool_calls
                        openai_messages[-1]["content"] = None
                    else:
                        openai_messages.append(
                            {
                                "role": "assistant",
                                "content": None,
                                "tool_calls": assistant_tool_calls,
                            }
                        )

    # 规范化最终消息
    final_openai_messages = []
    for msg_dict in openai_messages:
        if (
            msg_dict.get("role") == "assistant"
            and msg_dict.get("tool_calls")
            and "content" in msg_dict
            and not msg_dict["content"]
        ):
            warning(
                LogRecord(
                    event=LogEvent.MESSAGE_FORMAT_NORMALIZED.value,
                    message="Removing empty 'content' from assistant message with tool_calls.",
                    request_id=request_id,
                    data={"original_content": msg_dict["content"]},
                )
            )
            del msg_dict["content"]
        final_openai_messages.append(msg_dict)

    return final_openai_messages


def convert_anthropic_tools_to_openai(
    anthropic_tools: Optional[List[Tool]], target_model_name: str
) -> Optional[List[Dict[str, Any]]]:
    """
    将Anthropic工具定义转换为OpenAI函数定义，支持缓存控制。
    
    Args:
        anthropic_tools: Anthropic工具列表
        target_model_name: 目标模型名称
        
    Returns:
        OpenAI工具列表，如果输入为None则返回None
    """
    if not anthropic_tools:
        return None

    if settings.open_claude_cache and "claude" in target_model_name.lower():
        is_claude_model = True
    else:
        is_claude_model = False
    openai_tools = []
    for t in anthropic_tools:
        tool_dict = {
            "type": "function",
            "function": {
                "name": t.name,
                "description": t.description or "",
                "parameters": t.input_schema,
            },
        }
        if is_claude_model and t.cache_control:
            tool_dict["cache_control"] = t.cache_control.model_dump()
        openai_tools.append(tool_dict)

    return openai_tools


def convert_anthropic_tool_choice_to_openai(
    anthropic_choice: Optional[ToolChoice],
    request_id: Optional[str] = None,
) -> Optional[Union[str, Dict[str, Any]]]:
    """
    将Anthropic工具选择转换为OpenAI函数调用控制。
    
    Args:
        anthropic_choice: Anthropic工具选择
        request_id: 请求ID，用于日志记录
        
    Returns:
        OpenAI工具选择，如果输入为None则返回None
    """
    if not anthropic_choice:
        return None
    if anthropic_choice.type == "auto":
        return "auto"
    if anthropic_choice.type == "any":
        warning(
            LogRecord(
                event=LogEvent.TOOL_CHOICE_UNSUPPORTED.value,
                message="Anthropic tool_choice type 'any' mapped to OpenAI 'auto'. Exact behavior might differ (OpenAI 'auto' allows no tool use).",
                request_id=request_id,
                data={"anthropic_tool_choice": anthropic_choice.model_dump()},
            )
        )
        return "auto"
    if anthropic_choice.type == "tool" and anthropic_choice.name:
        return {"type": "function", "function": {"name": anthropic_choice.name}}

    warning(
        LogRecord(
            event=LogEvent.TOOL_CHOICE_UNSUPPORTED.value,
            message=f"Unsupported Anthropic tool_choice: {anthropic_choice.model_dump()}. Defaulting to 'auto'.",
            request_id=request_id,
            data={"anthropic_tool_choice": anthropic_choice.model_dump()},
        )
    )
    return "auto"


def convert_openai_to_anthropic_response(
    openai_response: Any,  # 使用Any类型以避免直接依赖openai.types.chat.ChatCompletion
    original_anthropic_model_name: str,
    request_id: Optional[str] = None,
) -> MessagesResponse:
    """
    将OpenAI响应转换为Anthropic响应格式。
    
    Args:
        openai_response: OpenAI响应对象
        original_anthropic_model_name: 原始Anthropic模型名称
        request_id: 请求ID，用于日志记录
        
    Returns:
        Anthropic格式的响应
    """
    anthropic_content: List[ContentBlock] = []
    anthropic_stop_reason: Optional[
        Literal["end_turn", "max_tokens", "stop_sequence", "tool_use", "error"]
    ] = None

    stop_reason_map: Dict[Optional[str], Optional[
        Literal["end_turn", "max_tokens", "stop_sequence", "tool_use", "error"]
    ]] = {
        "stop": "end_turn",
        "length": "max_tokens",
        "tool_calls": "tool_use",
        "function_call": "tool_use",
        "content_filter": "stop_sequence",
        None: "end_turn",
    }

    if openai_response.choices:
        choice = openai_response.choices[0]
        message = choice.message
        finish_reason = choice.finish_reason

        anthropic_stop_reason = stop_reason_map.get(finish_reason, "end_turn")

        if message.content:
            anthropic_content.append(
                ContentBlockText(type="text", text=message.content)
            )

        if message.tool_calls:
            for call in message.tool_calls:
                if call.type == "function":
                    tool_input_dict: Dict[str, Any] = {}
                    try:
                        parsed_input = json.loads(call.function.arguments)
                        if isinstance(parsed_input, dict):
                            tool_input_dict = parsed_input
                        else:
                            tool_input_dict = {"value": parsed_input}
                            warning(
                                LogRecord(
                                    event=LogEvent.TOOL_ARGS_TYPE_MISMATCH.value,
                                    message=f"OpenAI tool arguments for '{call.function.name}' parsed to non-dict type '{type(parsed_input).__name__}'. Wrapped in 'value'.",
                                    request_id=request_id,
                                    data={
                                        "tool_name": call.function.name,
                                        "tool_id": call.id,
                                    },
                                )
                            )
                    except json.JSONDecodeError as e:
                        error(
                            LogRecord(
                                event=LogEvent.TOOL_ARGS_PARSE_FAILURE.value,
                                message=f"Failed to parse JSON arguments for tool '{call.function.name}'. Storing raw string.",
                                request_id=request_id,
                                data={
                                    "tool_name": call.function.name,
                                    "tool_id": call.id,
                                    "raw_args": call.function.arguments,
                                },
                            ),
                            exc=e,
                        )
                        tool_input_dict = {
                            "error_parsing_arguments": call.function.arguments
                        }

                    anthropic_content.append(
                        ContentBlockToolUse(
                            type="tool_use",
                            id=call.id,
                            name=call.function.name,
                            input=tool_input_dict,
                        )
                    )
            if finish_reason == "tool_calls":
                anthropic_stop_reason = "tool_use"

    if not anthropic_content:
        anthropic_content.append(ContentBlockText(type="text", text=""))

    usage = openai_response.usage
    anthropic_usage = Usage(
        input_tokens=usage.prompt_tokens if usage else 0,
        output_tokens=usage.completion_tokens if usage else 0,
    )

    response_id = (
        f"msg_{openai_response.id}"
        if openai_response.id
        else f"msg_{request_id}_completed"
    )

    return MessagesResponse(
        id=response_id,
        type="message",
        role="assistant",
        model=original_anthropic_model_name,
        content=anthropic_content,
        stop_reason=anthropic_stop_reason,
        usage=anthropic_usage,
    )


def get_anthropic_error_details_from_exc(
    exc: Exception,
) -> Tuple[AnthropicErrorType, str, int, Optional[ProviderErrorMetadata]]:
    """
    将捕获的异常映射到Anthropic错误类型、消息、状态码和提供者详情。
    
    Args:
        exc: 捕获的异常
        
    Returns:
        错误类型、错误消息、HTTP状态码和提供者错误元数据的元组
    """
    from openai import (APIError, AuthenticationError, BadRequestError,
                        NotFoundError, PermissionDeniedError, RateLimitError,
                        UnprocessableEntityError)
    from models.errors import STATUS_CODE_ERROR_MAP, AnthropicErrorType
    
    error_type = AnthropicErrorType.API_ERROR
    error_message = str(exc)
    status_code = 500
    provider_details: Optional[ProviderErrorMetadata] = None

    if isinstance(exc, APIError):
        error_message = exc.message or str(exc)
        status_code = exc.status_code or 500
        error_type = STATUS_CODE_ERROR_MAP.get(
            status_code, AnthropicErrorType.API_ERROR
        )

        if hasattr(exc, "body") and isinstance(exc.body, dict):
            actual_error_details = exc.body.get("error", exc.body)
            provider_details = extract_provider_error_details(actual_error_details)

    if isinstance(exc, AuthenticationError):
        error_type = AnthropicErrorType.AUTHENTICATION
    elif isinstance(exc, RateLimitError):
        error_type = AnthropicErrorType.RATE_LIMIT
    elif isinstance(exc, (BadRequestError, UnprocessableEntityError)):
        error_type = AnthropicErrorType.INVALID_REQUEST
    elif isinstance(exc, PermissionDeniedError):
        error_type = AnthropicErrorType.PERMISSION
    elif isinstance(exc, NotFoundError):
        error_type = AnthropicErrorType.NOT_FOUND

    return error_type, error_message, status_code, provider_details


def format_anthropic_error_sse_event(
    error_type: AnthropicErrorType,
    message: str,
    provider_details: Optional[ProviderErrorMetadata] = None,
) -> str:
    """
    将错误格式化为Anthropic SSE 'error'事件结构。
    
    Args:
        error_type: Anthropic错误类型
        message: 错误消息
        provider_details: 提供者错误元数据
        
    Returns:
        格式化的SSE事件字符串
    """
    from models.errors import AnthropicErrorDetail, AnthropicErrorResponse
    
    anthropic_err_detail = AnthropicErrorDetail(type=error_type, message=message)
    if provider_details:
        anthropic_err_detail.provider = provider_details.provider_name
        if provider_details.raw_error and isinstance(
            provider_details.raw_error.get("error"), dict
        ):
            prov_err_obj = provider_details.raw_error["error"]
            anthropic_err_detail.provider_message = prov_err_obj.get("message")
            anthropic_err_detail.provider_code = prov_err_obj.get("code")
        elif provider_details.raw_error and isinstance(
            provider_details.raw_error.get("message"), str
        ):
            anthropic_err_detail.provider_message = provider_details.raw_error.get(
                "message"
            )
            anthropic_err_detail.provider_code = provider_details.raw_error.get("code")

    error_response = AnthropicErrorResponse(error=anthropic_err_detail)
    return f"event: error\ndata: {error_response.model_dump_json()}\n\n"


def select_target_model(client_model_name: str, request_id: str, settings) -> str:
    """
    根据客户端请求选择目标OpenRouter模型。
    
    Args:
        client_model_name: 客户端请求的模型名称
        request_id: 请求ID，用于日志记录
        settings: 应用程序设置
        
    Returns:
        目标模型名称
    """
    client_model_lower = client_model_name.lower()
    target_model: str

    if "opus" in client_model_lower or "sonnet" in client_model_lower:
        target_model = settings.big_model_name
    elif "haiku" in client_model_lower:
        target_model = settings.small_model_name
    else:
        target_model = settings.small_model_name
        warning(
            LogRecord(
                event=LogEvent.MODEL_SELECTION.value,
                message=f"Unknown client model '{client_model_name}', defaulting to SMALL model '{target_model}'.",
                request_id=request_id,
                data={
                    "client_model": client_model_name,
                    "default_target_model": target_model,
                },
            )
        )

    debug(
        LogRecord(
            event=LogEvent.MODEL_SELECTION.value,
            message=f"Client model '{client_model_name}' mapped to target model '{target_model}'.",
            request_id=request_id,
            data={"client_model": client_model_name, "target_model": target_model},
        )
    )
    return target_model
