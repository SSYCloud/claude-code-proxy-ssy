#!/bin/bash

# Claude Code Proxy Build Script
# This script builds binaries for multiple platforms

set -e

APP_NAME="claudeproxy"
VERSION="1.0.0"
BUILD_DIR="dist"
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

# Platforms to build for
declare -a PLATFORMS=(
    "linux/amd64"
    "linux/arm64" 
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/arm64"
)

echo "🔨 开始构建 ${APP_NAME} v${VERSION}..."

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    output_name="${APP_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        output_name+='.exe'
    fi
    
    echo "📦 构建 ${GOOS}/${GOARCH}..."
    
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

# Create release archives
echo ""
echo "📦 创建发布包..."

cd ${BUILD_DIR}

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r GOOS GOARCH <<< "$platform"
    
    binary_name="${APP_NAME}-${VERSION}-${GOOS}-${GOARCH}"
    if [ $GOOS = "windows" ]; then
        binary_name+='.exe'
    fi
    
    if [ -f "$binary_name" ]; then
        if [ $GOOS = "windows" ]; then
            zip_name="${APP_NAME}-${VERSION}-${GOOS}-${GOARCH}.zip"
            zip -q $zip_name $binary_name
            echo "✅ 创建: $zip_name"
        else
            tar_name="${APP_NAME}-${VERSION}-${GOOS}-${GOARCH}.tar.gz"
            tar -czf $tar_name $binary_name
            echo "✅ 创建: $tar_name"
        fi
    fi
done

cd ..

echo ""
echo "🎉 构建完成！"
echo "📁 构建文件位置: ${BUILD_DIR}/"
echo ""
echo "可用的二进制文件:"
ls -la ${BUILD_DIR}/${APP_NAME}-*

echo ""
echo "📋 安装说明:"
echo "1. 选择适合您操作系统的二进制文件"
echo "2. 解压并将二进制文件放到PATH路径中"
echo "3. 运行 'claudeproxy setup' 进行初始化配置"
echo "4. 运行 'claudeproxy start' 启动服务"
