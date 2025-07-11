#!/bin/bash

# Claude Code Proxy Installation Script
# This script automatically downloads and installs the latest version

set -e

REPO="SSYCloud/claude-code-proxy-ssy"
APP_NAME="claudeproxy"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Claude Code Proxy 安装脚本${NC}"
echo ""

# Detect OS and architecture
OS=$(uname -s)
ARCH=$(uname -m)

echo -e "${YELLOW}检测到系统: ${OS} ${ARCH}${NC}"

# Map architecture names to GitHub release naming
case $ARCH in
    "x86_64")
        RELEASE_ARCH="x86_64"
        ;;
    "arm64"|"aarch64")
        RELEASE_ARCH="arm64"
        ;;
    *)
        echo -e "${RED}❌ 不支持的架构: ${ARCH}${NC}"
        exit 1
        ;;
esac

# Map OS names to GitHub release naming
case $OS in
    "Linux")
        RELEASE_OS="Linux"
        BINARY_NAME="${APP_NAME}_${RELEASE_OS}_${RELEASE_ARCH}"
        ;;
    "Darwin")
        RELEASE_OS="Darwin"
        BINARY_NAME="${APP_NAME}_${RELEASE_OS}_${RELEASE_ARCH}"
        ;;
    *)
        echo -e "${RED}❌ 不支持的操作系统: ${OS}${NC}"
        echo "请手动下载适合您系统的二进制文件："
        echo "https://github.com/${REPO}/releases/latest"
        exit 1
        ;;
esac

echo -e "${YELLOW}目标文件: ${BINARY_NAME}${NC}"

# Download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_NAME}"

echo -e "${YELLOW}下载地址: ${DOWNLOAD_URL}${NC}"

# Create temporary directory
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/${APP_NAME}"

echo -e "${YELLOW}📥 正在下载...${NC}"

# Download the binary
if command -v curl &> /dev/null; then
    curl -L -o "$TMP_FILE" "$DOWNLOAD_URL"
elif command -v wget &> /dev/null; then
    wget -O "$TMP_FILE" "$DOWNLOAD_URL"
else
    echo -e "${RED}❌ 需要 curl 或 wget 来下载文件${NC}"
    exit 1
fi

# Check if download was successful
if [ ! -f "$TMP_FILE" ]; then
    echo -e "${RED}❌ 下载失败${NC}"
    exit 1
fi

# Make binary executable
chmod +x "$TMP_FILE"

# Install binary
echo -e "${YELLOW}📦 正在安装到 ${INSTALL_DIR}...${NC}"

if [ -w "$INSTALL_DIR" ]; then
    # Can write to install directory
    mv "$TMP_FILE" "${INSTALL_DIR}/${APP_NAME}"
else
    # Need sudo
    echo -e "${YELLOW}需要管理员权限来安装到 ${INSTALL_DIR}${NC}"
    sudo mv "$TMP_FILE" "${INSTALL_DIR}/${APP_NAME}"
fi

# Clean up
rm -rf "$TMP_DIR"

echo -e "${GREEN}✅ 安装完成！${NC}"
echo ""
echo -e "${GREEN}🎉 Claude Code Proxy 已成功安装${NC}"
echo ""
echo -e "${YELLOW}📋 下一步操作:${NC}"
echo "1. 配置 API 密钥:"
echo "   ${APP_NAME} setup"
echo ""
echo "2. 启动服务:"
echo "   ${APP_NAME} start"
echo ""
echo "3. 查看帮助:"
echo "   ${APP_NAME} --help"
echo ""
echo -e "${YELLOW}🔗 更多信息:${NC}"
echo "   项目地址: https://github.com/${REPO}"
echo "   使用文档: https://github.com/${REPO}/blob/main/README.md"
echo "   问题反馈: https://github.com/${REPO}/issues"
