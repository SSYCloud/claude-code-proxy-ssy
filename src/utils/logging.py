"""
日志模块：提供结构化日志记录功能
"""

import dataclasses
import enum
import json
import logging
import os
import traceback
from datetime import datetime, timezone
from logging.config import dictConfig
from typing import Any, Dict, Optional, Tuple

from rich.console import Console
from rich.panel import Panel
from rich.rule import Rule
from rich.text import Text

from config.settings import settings

# 控制台输出对象
_console = Console()
_error_console = Console(stderr=True, style="bold red")


class LogEvent(enum.Enum):
    """日志事件类型枚举，用于分类不同的日志事件"""
    MODEL_SELECTION = "model_selection"
    REQUEST_START = "request_start"
    REQUEST_COMPLETED = "request_completed"
    REQUEST_FAILURE = "request_failure"
    ANTHROPIC_REQUEST = "anthropic_body"
    OPENAI_REQUEST = "openai_request"
    OPENAI_RESPONSE = "openai_response"
    ANTHROPIC_RESPONSE = "anthropic_response"
    STREAMING_REQUEST = "streaming_request"
    STREAM_INTERRUPTED = "stream_interrupted"
    TOKEN_COUNT = "token_count"
    TOKEN_ENCODER_LOAD_FAILED = "token_encoder_load_failed"
    SYSTEM_PROMPT_ADJUSTED = "system_prompt_adjusted"
    TOOL_INPUT_SERIALIZATION_FAILURE = "tool_input_serialization_failure"
    IMAGE_FORMAT_UNSUPPORTED = "image_format_unsupported"
    MESSAGE_FORMAT_NORMALIZED = "message_format_normalized"
    TOOL_RESULT_SERIALIZATION_FAILURE = "tool_result_serialization_failure"
    TOOL_RESULT_PROCESSING = "tool_result_processing"
    TOOL_CHOICE_UNSUPPORTED = "tool_choice_unsupported"
    TOOL_ARGS_TYPE_MISMATCH = "tool_args_type_mismatch"
    TOOL_ARGS_PARSE_FAILURE = "tool_args_parse_failure"
    TOOL_ARGS_UNEXPECTED = "tool_args_unexpected"
    TOOL_ID_PLACEHOLDER = "tool_id_placeholder"
    TOOL_ID_UPDATED = "tool_id_updated"
    PARAMETER_UNSUPPORTED = "parameter_unsupported"
    HEALTH_CHECK = "health_check"
    PROVIDER_ERROR_DETAILS = "provider_error_details"


class JSONFormatter(logging.Formatter):
    """JSON格式的日志格式化器。"""
    def format(self, record: logging.LogRecord) -> str:
        header = {
            "timestamp": datetime.fromtimestamp(
                record.created, timezone.utc
            ).isoformat(),
            "level": record.levelname,
            "logger": record.name,
        }
        log_payload = getattr(record, "log_record", None)
        if isinstance(log_payload, LogRecord):
            header["detail"] = dataclasses.asdict(log_payload)
        else:
            header["message"] = record.getMessage()
            if record.exc_info:
                exc_type, exc_value, exc_tb = record.exc_info
                header["error"] = {
                    "name": exc_type.__name__ if exc_type else "UnknownError",
                    "message": str(exc_value),
                    "stack_trace": "".join(
                        traceback.format_exception(exc_type, exc_value, exc_tb)
                    ),
                    "args": exc_value.args if hasattr(exc_value, "args") else [],
                }
        return json.dumps(header, ensure_ascii=False)


class ConsoleJSONFormatter(JSONFormatter):
    """控制台JSON格式的日志格式化器。"""
    def format(self, record: logging.LogRecord) -> str:
        log_dict = json.loads(super().format(record))
        if (
            "detail" in log_dict
            and "error" in log_dict["detail"]
            and log_dict["detail"]["error"]
        ):
            if "stack_trace" in log_dict["detail"]["error"]:
                del log_dict["detail"]["error"]["stack_trace"]
        elif "error" in log_dict and log_dict["error"]:
            if "stack_trace" in log_dict["error"]:
                del log_dict["error"]["stack_trace"]
        return json.dumps(log_dict)


@dataclasses.dataclass
class LogError:
    """日志错误信息数据类"""
    name: str
    message: str
    stack_trace: Optional[str] = None
    args: Optional[Tuple[Any, ...]] = None


@dataclasses.dataclass
class LogRecord:
    """结构化日志记录数据类"""
    event: str
    message: str
    request_id: Optional[str] = None
    data: Optional[Dict[str, Any]] = None
    error: Optional[LogError] = None


# 配置日志系统
dictConfig(
    {
        "version": 1,
        "disable_existing_loggers": False,
        "formatters": {
            "json": {"()": JSONFormatter},
            "console_json": {"()": ConsoleJSONFormatter},
        },
        "handlers": {
            "default": {
                "class": "logging.StreamHandler",
                "formatter": "console_json",
                "stream": "ext://sys.stdout",
            },
        },
        "loggers": {
            "": {"handlers": ["default"], "level": "WARNING"},
            settings.app_name: {
                "handlers": ["default"],
                "level": settings.log_level.upper(),
                "propagate": False,
            },
            "uvicorn": {"handlers": ["default"], "level": "INFO", "propagate": False},
            "uvicorn.error": {
                "handlers": ["default"],
                "level": "INFO",
                "propagate": False,
            },
            "uvicorn.access": {
                "handlers": ["default"],
                "level": "INFO",
                "propagate": False,
            },
        },
    }
)

# 创建日志记录器
_logger = logging.getLogger(settings.app_name)
_request_logger = logging.getLogger(f"{settings.app_name}.requests")

# 配置文件日志（如果启用）
if settings.log_file_path:
    try:
        log_dir = os.path.dirname(settings.log_file_path)
        if log_dir:
            os.makedirs(log_dir, exist_ok=True)
        file_handler = logging.FileHandler(settings.log_file_path, mode="a")
        file_handler.setFormatter(JSONFormatter())
        _logger.addHandler(file_handler)
    except Exception as e:
        _error_console.print(
            f"Failed to configure file logging to {settings.log_file_path}: {e}"
        )


def _log(level: int, record: LogRecord, exc: Optional[Exception] = None) -> None:
    """记录日志的内部函数。"""
    if exc:
        record.error = LogError(
            name=type(exc).__name__,
            message=str(exc),
            stack_trace="".join(
                traceback.format_exception(type(exc), exc, exc.__traceback__)
            ),
            args=exc.args if hasattr(exc, "args") else tuple(),
        )
        if not record.message and str(exc):
            record.message = str(exc)
        elif not record.message:
            record.message = "An unspecified error occurred"

    _logger.log(level=level, msg=record.message, extra={"log_record": record})


def debug(record: LogRecord):
    """记录调试级别的日志。"""
    _log(logging.DEBUG, record)


def info(record: LogRecord):
    """记录信息级别的日志。"""
    _log(logging.INFO, record)


def warning(record: LogRecord, exc: Optional[Exception] = None):
    """记录警告级别的日志。"""
    _log(logging.WARNING, record, exc=exc)


def error(record: LogRecord, exc: Optional[Exception] = None):
    """记录错误级别的日志。"""
    if exc:
        _error_console.print_exception(show_locals=False, width=120)
    _log(logging.ERROR, record, exc=exc)


def critical(record: LogRecord, exc: Optional[Exception] = None):
    """记录严重错误级别的日志。"""
    _log(logging.CRITICAL, record, exc=exc)


def print_startup_banner():
    """打印应用程序启动横幅"""
    _console.print(
        r"""[bold blue]
           /$$                           /$$
          | $$                          | $$
  /$$$$$$$| $$  /$$$$$$  /$$   /$$  /$$$$$$$  /$$$$$$         /$$$$$$   /$$$$$$   /$$$$$$  /$$   /$$ /$$   /$$
 /$$_____/| $$ |____  $$| $$  | $$ /$$__  $$ /$$__  $$       /$$__  $$ /$$__  $$ /$$__  $$|  $$ /$$/| $$  | $$
| $$      | $$  /$$$$$$$| $$  | $$| $$  | $$| $$$$$$$$      | $$  \ $$| $$  \__/| $$  \ $$ \  $$$$/ | $$  | $$
| $$      | $$ /$$__  $$| $$  | $$| $$  | $$| $$_____/      | $$  | $$| $$      | $$  | $$  >$$  $$ | $$  | $$
|  $$$$$$$| $$|  $$$$$$$|  $$$$$$/|  $$$$$$$|  $$$$$$$      | $$$$$$$/| $$      |  $$$$$$/ /$$/\  $$|  $$$$$$$
 \_______/|__/ \_______/ \______/  \_______/ \_______/      | $$____/ |__/       \______/ |__/  \__/ \____  $$
                                                            | $$                                     /$$  | $$
                                                            | $$                                    |  $$$$$$/
                                                            |__/                                     \______/ 
    [/]""",
        justify="left",
    )
    config_details_text = Text.assemble(
        ("   Version       : ", "default"),
        (f"v{settings.app_version}", "bold cyan"),
        ("\n   Big Model     : ", "default"),
        (settings.big_model_name, "magenta"),
        ("\n   Small Model   : ", "default"),
        (settings.small_model_name, "green"),
        ("\n   Log Level     : ", "default"),
        (settings.log_level.upper(), "yellow"),
        ("\n   Log File      : ", "default"),
        (settings.log_file_path or "Disabled", "dim"),
        ("\n   Listening on  : ", "default"),
        (f"http://{settings.host}:{settings.port}", "bold white"),
        ("\n   Reload        : ", "default"),
        ("Enabled", "bold orange1") if settings.reload else ("Disabled", "dim"),
    )
    _console.print(
        Panel(
            config_details_text,
            title="Anthropic Proxy Configuration",
            border_style="blue",
            expand=False,
        )
    )
    _console.print(Rule("Starting Uvicorn server...", style="dim blue"))
