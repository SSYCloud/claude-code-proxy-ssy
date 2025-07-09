"""
流式处理服务模块：处理OpenAI流式响应到Anthropic SSE事件的转换
"""

import json
import time
import uuid
from typing import Any, AsyncGenerator, Dict, List, Literal, Optional

import tiktoken
from openai import AsyncStream
from openai.types.chat import ChatCompletionChunk

from models.anthropic import ContentBlockText, ContentBlockToolUse
from services.conversion import (format_anthropic_error_sse_event,
                                    get_anthropic_error_details_from_exc)
from services.token_counting import get_token_encoder
from utils.logging import LogEvent, LogRecord, debug, error, warning


# 停止原因映射
StopReasonType = Optional[
    Literal["end_turn", "max_tokens", "stop_sequence", "tool_use", "error"]
]


async def handle_anthropic_streaming_response_from_openai_stream(
    openai_stream: AsyncStream[ChatCompletionChunk],
    original_anthropic_model_name: str,
    estimated_input_tokens: int,
    request_id: str,
    start_time_mono: float,
) -> AsyncGenerator[str, None]:
    """
    消费OpenAI流并生成Anthropic兼容的SSE事件。
    修复：正确处理混合文本/工具使用的内容块索引。
    
    Args:
        openai_stream: OpenAI流式响应
        original_anthropic_model_name: 原始Anthropic模型名称
        estimated_input_tokens: 估计的输入令牌数
        request_id: 请求ID
        start_time_mono: 开始时间（单调时钟）
        
    Yields:
        Anthropic兼容的SSE事件字符串
    """
    # 生成唯一的消息ID
    anthropic_message_id = f"msg_stream_{request_id}_{uuid.uuid4().hex[:8]}"

    next_anthropic_block_idx = 0
    text_block_anthropic_idx: Optional[int] = None

    # 跟踪OpenAI工具索引到Anthropic块索引的映射
    openai_tool_idx_to_anthropic_block_idx: Dict[int, int] = {}

    # 跟踪工具状态
    tool_states: Dict[int, Dict[str, Any]] = {}

    # 跟踪已发送的工具块开始事件
    sent_tool_block_starts: set[int] = set()

    output_token_count = 0
    final_anthropic_stop_reason: StopReasonType = None

    # 获取令牌编码器
    enc = get_token_encoder(original_anthropic_model_name, request_id)

    # OpenAI到Anthropic停止原因映射
    openai_to_anthropic_stop_reason_map: Dict[Optional[str], StopReasonType] = {
        "stop": "end_turn",
        "length": "max_tokens",
        "tool_calls": "tool_use",
        "function_call": "tool_use",
        "content_filter": "stop_sequence",
        None: None,
    }

    stream_status_code = 200
    stream_final_message = "Streaming request completed successfully."
    stream_log_event = LogEvent.REQUEST_COMPLETED.value

    try:
        # 发送消息开始事件
        message_start_event_data = {
            "type": "message_start",
            "message": {
                "id": anthropic_message_id,
                "type": "message",
                "role": "assistant",
                "model": original_anthropic_model_name,
                "content": [],
                "stop_reason": None,
                "stop_sequence": None,
                "usage": {"input_tokens": estimated_input_tokens, "output_tokens": 0},
            },
        }
        yield f"event: message_start\ndata: {json.dumps(message_start_event_data)}\n\n"
        yield f"event: ping\ndata: {json.dumps({'type': 'ping'})}\n\n"

        # 处理流式响应
        async for chunk in openai_stream:
            if not chunk.choices:
                continue

            delta = chunk.choices[0].delta
            openai_finish_reason = chunk.choices[0].finish_reason

            # 处理文本内容
            if delta.content:
                output_token_count += len(enc.encode(delta.content))
                if text_block_anthropic_idx is None:
                    text_block_anthropic_idx = next_anthropic_block_idx
                    next_anthropic_block_idx += 1
                    start_text_event = {
                        "type": "content_block_start",
                        "index": text_block_anthropic_idx,
                        "content_block": {"type": "text", "text": ""},
                    }
                    yield f"event: content_block_start\ndata: {json.dumps(start_text_event)}\n\n"

                text_delta_event = {
                    "type": "content_block_delta",
                    "index": text_block_anthropic_idx,
                    "delta": {"type": "text_delta", "text": delta.content},
                }
                yield f"event: content_block_delta\ndata: {json.dumps(text_delta_event)}\n\n"

            # 处理工具调用
            if delta.tool_calls:
                for tool_delta in delta.tool_calls:
                    openai_tc_idx = tool_delta.index

                    # 如果是新的工具调用，创建新的块索引
                    if openai_tc_idx not in openai_tool_idx_to_anthropic_block_idx:
                        current_anthropic_tool_block_idx = next_anthropic_block_idx
                        next_anthropic_block_idx += 1
                        openai_tool_idx_to_anthropic_block_idx[openai_tc_idx] = (
                            current_anthropic_tool_block_idx
                        )

                        # 初始化工具状态
                        tool_states[current_anthropic_tool_block_idx] = {
                            "id": tool_delta.id
                            or f"tool_ph_{request_id}_{current_anthropic_tool_block_idx}",
                            "name": "",
                            "arguments_buffer": "",
                        }
                        if not tool_delta.id:
                            warning(
                                LogRecord(
                                    LogEvent.TOOL_ID_PLACEHOLDER.value,
                                    f"Generated placeholder Tool ID for OpenAI tool index {openai_tc_idx} -> Anthropic block {current_anthropic_tool_block_idx}",
                                    request_id,
                                )
                            )
                    else:
                        current_anthropic_tool_block_idx = (
                            openai_tool_idx_to_anthropic_block_idx[openai_tc_idx]
                        )

                    tool_state = tool_states[current_anthropic_tool_block_idx]

                    # 更新工具ID（如果有）
                    if tool_delta.id and tool_state["id"].startswith("tool_ph_"):
                        debug(
                            LogRecord(
                                LogEvent.TOOL_ID_UPDATED.value,
                                f"Updated placeholder Tool ID for Anthropic block {current_anthropic_tool_block_idx} to {tool_delta.id}",
                                request_id,
                            )
                        )
                        tool_state["id"] = tool_delta.id

                    # 更新工具函数信息
                    if tool_delta.function:
                        if tool_delta.function.name:
                            tool_state["name"] = tool_delta.function.name
                        if tool_delta.function.arguments:
                            tool_state["arguments_buffer"] += (
                                tool_delta.function.arguments
                            )
                            output_token_count += len(
                                enc.encode(tool_delta.function.arguments)
                            )

                    # 发送工具块开始事件（如果尚未发送且有足够信息）
                    if (
                        current_anthropic_tool_block_idx not in sent_tool_block_starts
                        and tool_state["id"]
                        and not tool_state["id"].startswith("tool_ph_")
                        and tool_state["name"]
                    ):
                        start_tool_event = {
                            "type": "content_block_start",
                            "index": current_anthropic_tool_block_idx,
                            "content_block": {
                                "type": "tool_use",
                                "id": tool_state["id"],
                                "name": tool_state["name"],
                                "input": {},
                            },
                        }
                        yield f"event: content_block_start\ndata: {json.dumps(start_tool_event)}\n\n"
                        sent_tool_block_starts.add(current_anthropic_tool_block_idx)

                    # 发送参数增量事件
                    if (
                        tool_delta.function
                        and tool_delta.function.arguments
                        and current_anthropic_tool_block_idx in sent_tool_block_starts
                    ):
                        args_delta_event = {
                            "type": "content_block_delta",
                            "index": current_anthropic_tool_block_idx,
                            "delta": {
                                "type": "input_json_delta",
                                "partial_json": tool_delta.function.arguments,
                            },
                        }
                        yield f"event: content_block_delta\ndata: {json.dumps(args_delta_event)}\n\n"

            # 处理完成原因
            if openai_finish_reason:
                final_anthropic_stop_reason = openai_to_anthropic_stop_reason_map.get(
                    openai_finish_reason, "end_turn"
                )
                if openai_finish_reason == "tool_calls":
                    final_anthropic_stop_reason = "tool_use"
                break

        # 发送内容块停止事件
        if text_block_anthropic_idx is not None:
            yield f"event: content_block_stop\ndata: {json.dumps({'type': 'content_block_stop', 'index': text_block_anthropic_idx})}\n\n"

        # 发送工具块停止事件
        for anthropic_tool_idx in sent_tool_block_starts:
            tool_state_to_finalize = tool_states.get(anthropic_tool_idx)
            if tool_state_to_finalize:
                try:
                    json.loads(tool_state_to_finalize["arguments_buffer"])
                except json.JSONDecodeError:
                    warning(
                        LogRecord(
                            event=LogEvent.TOOL_ARGS_PARSE_FAILURE.value,
                            message=f"Buffered arguments for tool '{tool_state_to_finalize.get('name')}' (Anthropic block {anthropic_tool_idx}) did not form valid JSON.",
                            request_id=request_id,
                            data={
                                "buffered_args": tool_state_to_finalize[
                                    "arguments_buffer"
                                ][:100]
                            },
                        )
                    )
            yield f"event: content_block_stop\ndata: {json.dumps({'type': 'content_block_stop', 'index': anthropic_tool_idx})}\n\n"

        # 设置默认停止原因（如果未设置）
        if final_anthropic_stop_reason is None:
            final_anthropic_stop_reason = "end_turn"

        # 发送消息增量事件
        message_delta_event = {
            "type": "message_delta",
            "delta": {
                "stop_reason": final_anthropic_stop_reason,
                "stop_sequence": None,
            },
            "usage": {"output_tokens": output_token_count},
        }
        yield f"event: message_delta\ndata: {json.dumps(message_delta_event)}\n\n"
        yield f"event: message_stop\ndata: {json.dumps({'type': 'message_stop'})}\n\n"

    except Exception as e:
        # 处理异常
        stream_status_code = 500
        stream_log_event = LogEvent.REQUEST_FAILURE.value
        error_type, error_msg_str, _, provider_err_details = (
            get_anthropic_error_details_from_exc(e)
        )
        stream_final_message = f"Error during OpenAI stream conversion: {error_msg_str}"
        final_anthropic_stop_reason = "error"

        error(
            LogRecord(
                event=LogEvent.STREAM_INTERRUPTED.value,
                message=stream_final_message,
                request_id=request_id,
                data={
                    "error_type": error_type.value,
                    "provider_details": provider_err_details.model_dump()
                    if provider_err_details
                    else None,
                },
            ),
            exc=e,
        )
        yield format_anthropic_error_sse_event(
            error_type, error_msg_str, provider_err_details
        )

    finally:
        # 记录流式请求完成
        duration_ms = (time.monotonic() - start_time_mono) * 1000
        log_data = {
            "status_code": stream_status_code,
            "duration_ms": duration_ms,
            "input_tokens": estimated_input_tokens,
            "output_tokens": output_token_count,
            "stop_reason": final_anthropic_stop_reason,
        }
        if stream_log_event == LogEvent.REQUEST_COMPLETED.value:
            debug(
                LogRecord(
                    event=stream_log_event,
                    message=stream_final_message,
                    request_id=request_id,
                    data=log_data,
                )
            )
        else:
            error(
                LogRecord(
                    event=stream_log_event,
                    message=stream_final_message,
                    request_id=request_id,
                    data=log_data,
                )
            )
