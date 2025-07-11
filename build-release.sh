#!/bin/bash

# Claude Code Proxy Release Build Script
# This script builds binaries for GitHub releases with proper naming

set -e

APP_NAME="claudeproxy"
VERSION="${1:-v0.1.0}"  # Use provided version or default to v0.1.0
BUILD_DIR="release"
MAIN_FILE="main.go"

# Clean previous builds
echo "🧹 清理之前的构建..."
rm -rf ${BUILD_DIR}
mkdir -p ${BUILD_DIR}

# Get commit hash for version info
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS="-X main.version=${VERSION} -X main.commit=${COMMIT_HASH} -X main.buildTime=${BUILD_TIME} -s -w"

# Platforms to build for (using GitHub Release naming convention)
declare -a PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "🔨 开始构建 ${APP_NAME} ${VERSION} for GitHub Release..."

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    # GitHub Release naming convention: claudeproxy_Linux_x86_64
    case $GOOS in
        "linux")
            OS_NAME="Linux"
            ;;
        "darwin")
            OS_NAME="Darwin"
            ;;
        "windows")
            OS_NAME="Windows"
            ;;
    esac
    
    case $GOARCH in
        "amd64")
            ARCH_NAME="x86_64"
            ;;
        "arm64")
            ARCH_NAME="arm64"
            ;;
    esac
    
    output_name="${APP_NAME}_${OS_NAME}_${ARCH_NAME}"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "📦 构建 ${GOOS}/${GOARCH} -> ${output_name}..."
    
    env GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build \
        -ldflags="${LDFLAGS}" \
        -o ${BUILD_DIR}/${output_name} \
        ${MAIN_FILE}
    
    if [ $? -ne 0 ]; then
        echo "❌ 构建 ${GOOS}/${GOARCH} 失败"
        exit 1
    fi
    
    echo "✅ 构建完成: ${BUILD_DIR}/${output_name}"
done

echo ""
echo "🎉 GitHub Release 构建完成！"
echo "📁 构建文件位置: ${BUILD_DIR}/"
echo ""
echo "可用的二进制文件:"
ls -la ${BUILD_DIR}/${APP_NAME}_*

echo ""
echo "📋 下一步操作:"
echo "1. 运行 './create-release.sh ${VERSION}' 创建 GitHub Release"
echo "2. 或者手动上传这些文件到 GitHub Release 页面"
echo ""
echo "📝 用户安装命令示例:"
echo "# Linux/macOS:"
echo "sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_\$(uname -s)_\$(uname -m)"
echo "sudo chmod +x /usr/local/bin/claudeproxy"
echo ""
echo "# Windows (PowerShell):"
echo "Invoke-WebRequest -Uri \"https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_Windows_x86_64.exe\" -OutFile \"claudeproxy.exe\""
