@echo off
setlocal enabledelayedexpansion

:: 设置颜色
set GREEN=92m
set YELLOW=93m
set RED=91m
set NC=0m

echo [%YELLOW%===== 启动 Claude Code Provider Proxy 服务 =====[%NC%

:: 检查环境变量文件是否存在
if not exist .env (
    echo [%YELLOW%环境变量文件 .env 不存在，将从 .env.example 创建[%NC%
    copy .env.example .env
    echo [%YELLOW%请编辑 .env 文件设置您的 API 密钥和其他配置[%NC%
    echo [%RED%按 Ctrl+C 退出并编辑 .env 文件，或按 Enter 继续[%NC%
    pause > nul
)

:: 检查虚拟环境
if not exist venv (
    echo [%YELLOW%未检测到虚拟环境，正在创建...[%NC%
    python -m venv venv
    echo [%GREEN%虚拟环境已创建[%NC%
)

:: 激活虚拟环境
echo [%YELLOW%激活虚拟环境...[%NC%
call venv\Scripts\activate.bat

:: 安装依赖
echo [%YELLOW%安装依赖...[%NC%
pip install -r requirements.txt

:: 清除可能存在的环境变量
echo [%YELLOW%清除环境变量...[%NC%
set OPENAI_API_KEY=
set BIG_MODEL_NAME=
set SMALL_MODEL_NAME=
set BASE_URL=
set REFERRER_URL=
set APP_NAME=
set APP_VERSION=
set LOG_LEVEL=
set LOG_FILE_PATH=
set HOST=
set PORT=
set RELOAD=
set OPEN_CLAUDE_CACHE=

:: 启动应用
echo [%GREEN%启动应用...[%NC%
cd src && python -m main

:: 捕获退出码
set EXIT_CODE=%ERRORLEVEL%

:: 如果应用异常退出，显示错误信息
if %EXIT_CODE% NEQ 0 (
    echo [%RED%应用异常退出，退出码: %EXIT_CODE%[%NC%
    echo [%YELLOW%请检查日志获取更多信息[%NC%
)

:: 退出虚拟环境
deactivate

exit /b %EXIT_CODE%
