@echo off
setlocal enabledelayedexpansion

:: 设置颜色
set GREEN=92m
set YELLOW=93m
set RED=91m
set NC=0m

echo [%YELLOW%===== 使用 Docker 启动 Claude Code Provider Proxy 服务 =====[%NC%

:: 检查环境变量文件是否存在
if not exist .env (
    echo [%YELLOW%环境变量文件 .env 不存在，将从 .env.example 创建[%NC%
    copy .env.example .env
    echo [%YELLOW%请编辑 .env 文件设置您的 API 密钥和其他配置[%NC%
    echo [%RED%按 Ctrl+C 退出并编辑 .env 文件，或按 Enter 继续[%NC%
    pause > nul
)

:: 检查 Docker 是否安装
where docker >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [%RED%Docker 未安装，请先安装 Docker[%NC%
    exit /b 1
)

:: 检查 Docker Compose 是否安装
where docker-compose >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo [%RED%Docker Compose 未安装，请先安装 Docker Compose[%NC%
    exit /b 1
)

:: 构建并启动容器
echo [%YELLOW%构建并启动 Docker 容器...[%NC%
docker-compose up --build -d

:: 检查容器是否成功启动
if %ERRORLEVEL% EQU 0 (
    echo [%GREEN%Docker 容器已成功启动[%NC%
    echo [%GREEN%服务运行在 http://localhost:8000[%NC%
    echo [%YELLOW%使用以下命令查看日志:[%NC%
    echo [%YELLOW%docker-compose logs -f[%NC%
    echo [%YELLOW%使用以下命令停止服务:[%NC%
    echo [%YELLOW%docker-compose down[%NC%
) else (
    echo [%RED%Docker 容器启动失败[%NC%
    echo [%YELLOW%使用以下命令查看错误日志:[%NC%
    echo [%YELLOW%docker-compose logs[%NC%
    exit /b 1
)

endlocal
