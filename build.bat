@echo off
REM Claude Code Proxy Build Script for Windows

set APP_NAME=claudeproxy
set VERSION=1.0.0
set BUILD_DIR=dist
set MAIN_FILE=main.go

echo 🧹 清理之前的构建...
if exist %BUILD_DIR% rmdir /s /q %BUILD_DIR%
mkdir %BUILD_DIR%

echo 🔨 开始构建 %APP_NAME% v%VERSION%...

REM Build for Windows AMD64
echo 📦 构建 windows/amd64...
set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\%APP_NAME%-windows-amd64.exe %MAIN_FILE%

REM Build for Windows ARM64
echo 📦 构建 windows/arm64...
set GOOS=windows
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\%APP_NAME%-windows-arm64.exe %MAIN_FILE%

REM Build for Linux AMD64
echo 📦 构建 linux/amd64...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\%APP_NAME%-linux-amd64 %MAIN_FILE%

REM Build for macOS AMD64
echo 📦 构建 darwin/amd64...
set GOOS=darwin
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\%APP_NAME%-darwin-amd64 %MAIN_FILE%

REM Build for macOS ARM64
echo 📦 构建 darwin/arm64...
set GOOS=darwin
set GOARCH=arm64
set CGO_ENABLED=0
go build -ldflags="-s -w" -o %BUILD_DIR%\%APP_NAME%-darwin-arm64 %MAIN_FILE%

echo.
echo 🎉 构建完成！
echo 📁 构建文件位置: %BUILD_DIR%\
echo.
echo 可用的二进制文件:
dir %BUILD_DIR%

echo.
echo 📋 安装说明:
echo 1. 选择适合您操作系统的二进制文件
echo 2. 将二进制文件放到PATH路径中
echo 3. 运行 'claudeproxy setup' 进行初始化配置
echo 4. 运行 'claudeproxy start' 启动服务

pause
