#!/bin/bash

# GitHub Release Creation Script
# This script creates a GitHub release and uploads binaries

set -e

VERSION="${1:-v0.1.0}"
REPO="SSYCloud/claude-code-proxy-ssy"
BUILD_DIR="release"
APP_NAME="claudeproxy"

echo "🚀 创建 GitHub Release ${VERSION}..."

# Check if gh CLI is installed
if ! command -v gh &> /dev/null; then
    echo "❌ GitHub CLI (gh) 未安装"
    echo "请先安装 GitHub CLI:"
    echo "  macOS: brew install gh"
    echo "  Linux: 参考 https://github.com/cli/cli/blob/trunk/docs/install_linux.md"
    echo "  Windows: 参考 https://github.com/cli/cli/releases"
    exit 1
fi

# Check if user is logged in to GitHub
if ! gh auth status &> /dev/null; then
    echo "❌ 未登录到 GitHub"
    echo "请先登录: gh auth login"
    exit 1
fi

# Check if build directory exists
if [ ! -d "${BUILD_DIR}" ]; then
    echo "❌ 构建目录 ${BUILD_DIR} 不存在"
    echo "请先运行: ./build-release.sh ${VERSION}"
    exit 1
fi

# Create release notes
RELEASE_NOTES="# Claude Code Proxy ${VERSION}

## 🎉 新特性
- 提供 OpenAI 兼容的 API 接口
- 支持 Claude 3.5 Sonnet 模型
- 支持流式响应
- 支持多种部署方式

## 📦 安装方式

### 快速安装（推荐）

**Linux/macOS:**
\`\`\`bash
sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/${REPO}/releases/latest/download/claudeproxy_\$(uname -s)_\$(uname -m)
sudo chmod +x /usr/local/bin/claudeproxy
\`\`\`

**Windows (PowerShell):**
\`\`\`powershell
Invoke-WebRequest -Uri \"https://github.com/${REPO}/releases/latest/download/claudeproxy_Windows_x86_64.exe\" -OutFile \"claudeproxy.exe\"
\`\`\`

### 手动下载

选择适合您系统的二进制文件：

- **Linux x86_64**: claudeproxy_Linux_x86_64
- **Linux ARM64**: claudeproxy_Linux_arm64
- **macOS Intel**: claudeproxy_Darwin_x86_64
- **macOS Apple Silicon**: claudeproxy_Darwin_arm64
- **Windows x86_64**: claudeproxy_Windows_x86_64.exe
- **Windows ARM64**: claudeproxy_Windows_arm64.exe

## 🚀 使用方法

1. 配置 API 密钥：
   \`\`\`bash
   claudeproxy setup
   \`\`\`

2. 启动服务：
   \`\`\`bash
   claudeproxy start
   \`\`\`

3. 使用 OpenAI 兼容的 API：
   \`\`\`bash
   curl -X POST http://localhost:8080/v1/chat/completions \\
     -H \"Content-Type: application/json\" \\
     -H \"Authorization: Bearer your-api-key\" \\
     -d '{
       \"model\": \"claude-3-5-sonnet-20241022\",
       \"messages\": [
         {\"role\": \"user\", \"content\": \"Hello!\"}
       ]
     }'
   \`\`\`

## 📋 系统要求

- 操作系统: Linux, macOS, Windows
- 架构: x86_64, ARM64
- 网络: 需要访问 Anthropic API

## 🔧 配置说明

详细配置说明请参考项目文档。

---

**完整源代码**: https://github.com/${REPO}
**问题反馈**: https://github.com/${REPO}/issues
**使用文档**: https://github.com/${REPO}/blob/main/README.md"

echo "📝 创建 Release Notes..."
echo "$RELEASE_NOTES" > release-notes.md

# Check if tag exists
if ! git tag -l | grep -q "^${VERSION}$"; then
    echo "📌 创建标签 ${VERSION}..."
    git tag ${VERSION}
    git push origin ${VERSION}
else
    echo "📌 标签 ${VERSION} 已存在"
fi

# Create GitHub release
echo "🎯 创建 GitHub Release..."
gh release create ${VERSION} \
    --repo ${REPO} \
    --title "Claude Code Proxy ${VERSION}" \
    --notes-file release-notes.md \
    --draft=false \
    --prerelease=false

# Upload binaries
echo "📤 上传二进制文件..."
for file in ${BUILD_DIR}/${APP_NAME}_*; do
    if [ -f "$file" ]; then
        filename=$(basename "$file")
        echo "  上传 $filename..."
        gh release upload ${VERSION} "$file" --repo ${REPO} --clobber
    fi
done

# Clean up
rm -f release-notes.md

echo ""
echo "🎉 GitHub Release 创建完成！"
echo "🔗 Release 链接: https://github.com/${REPO}/releases/tag/${VERSION}"
echo ""
echo "📋 用户可以使用以下命令安装:"
echo ""
echo "Linux/macOS:"
echo "sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/${REPO}/releases/latest/download/claudeproxy_\$(uname -s)_\$(uname -m)"
echo "sudo chmod +x /usr/local/bin/claudeproxy"
echo ""
echo "Windows (PowerShell):"
echo "Invoke-WebRequest -Uri \"https://github.com/${REPO}/releases/latest/download/claudeproxy_Windows_x86_64.exe\" -OutFile \"claudeproxy.exe\""
