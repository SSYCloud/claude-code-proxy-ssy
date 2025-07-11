# Claude Code Proxy SSY

Claude Code Proxy SSY 是一个命令行工具，可以将Claude API转换为胜算云格式，让您在Claude的应用程序中使用胜算云全球模型API。

## ✨ 功能特性

- 🚀 **简单易用**: 一键设置和启动
- 🔧 **交互式配置**: 引导式配置向导
- 🌐 **多平台支持**: 支持 Windows、macOS、Linux
- 🔄 **模型选择**: 支持选择不同的大小模型
- 📱 **后台运行**: 服务可在后台运行
- ⚙️ **配置管理**: 简单的配置修改和查看

## 📦 安装

### 方式一: 快速安装（推荐）

**Linux/macOS:**
```bash
sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_$(uname -s)_$(uname -m)
sudo chmod +x /usr/local/bin/claudeproxy
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_Windows_x86_64.exe" -OutFile "claudeproxy.exe"
```

### 方式二: 使用安装脚本

```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/claude-code-proxy-ssy/main/install.sh | bash
```

### 方式三: 手动下载

1. 从 [Releases](https://github.com/SSYCloud/claude-code-proxy-ssy/releases) 页面下载适合您操作系统的二进制文件
2. 解压并将文件放到系统 PATH 中
3. 运行 `claudeproxy setup` 进行初始化

支持的平台：
- **Linux x86_64**: claudeproxy_Linux_x86_64
- **Linux ARM64**: claudeproxy_Linux_arm64
- **macOS Intel**: claudeproxy_Darwin_x86_64
- **macOS Apple Silicon**: claudeproxy_Darwin_arm64
- **Windows x86_64**: claudeproxy_Windows_x86_64.exe
- **Windows ARM64**: claudeproxy_Windows_arm64.exe

### 方式四: 从源码构建

```bash
# 克隆仓库
git clone https://github.com/SSYCloud/claude-code-proxy-ssy.git
cd claude-code-proxy-ssy

# 构建当前平台
make build

# 或构建所有平台
make build-all

# 安装到本地 (仅 macOS/Linux)
make install
```

## 🚀 快速开始

### 1. 初始化配置

```bash
claudeproxy setup
```

这个命令会:
- 引导您输入胜算云 API 密钥
- 获取可用模型列表
- 让您选择大模型和小模型
- 保存配置到 `~/.claudeproxy/.env`

### 2. 启动服务

```bash
claudeproxy start
```

服务将在后台启动，默认监听 `http://127.0.0.1:3180`

**自动配置 Claude 环境变量**: 服务启动成功后，会自动设置以下环境变量，方便Claude Desktop等应用直接使用：

- `ANTHROPIC_BASE_URL=http://127.0.0.1:3180` (或您配置的HOST:PORT)
- `ANTHROPIC_AUTH_TOKEN=claudeproxy`

### 3. 使用服务

现在您可以将任何支持 OpenAI API 的应用程序配置为使用 `http://127.0.0.1:3180` 作为 API 端点。

对于Claude Desktop等原生支持Anthropic API的应用，环境变量已自动配置，无需额外设置。

## 📋 命令使用

### 基本命令

```bash
# 查看帮助
claudeproxy --help

# 初始化配置
claudeproxy setup

# 启动服务
claudeproxy start

# 停止服务
claudeproxy stop

# 查看服务状态
claudeproxy status

# 查看当前配置
claudeproxy config

# 修改配置
claudeproxy set

# 清除所有环境变量和配置
claudeproxy clean
```

### 配置修改

使用 `claudeproxy set` 命令可以:

- 修改 API 密钥
- 重新选择模型
- 查看当前配置
- 重新初始化配置

### 清理配置

使用 `claudeproxy clean` 命令可以完全清除所有项目相关的配置：

- 停止正在运行的服务
- 清除所有环境变量（包括ANTHROPIC_*变量，当前终端和全局环境）
- 删除配置文件
- 需要重启终端以确保环境变量完全清除

## ⚙️ 配置选项

默认配置保存在 `~/.claudeproxy/.env` 文件中:

```bash
BASE_URL=https://router.shengsuanyun.com/api/v1
REFERRER_URL=https://www.shengsuanyun.com
APP_NAME=ClaudeCodeProxy
APP_VERSION=1.0.0
HOST=127.0.0.1
PORT=3180
RELOAD=true
OPEN_CLAUDE_CACHE=true
LOG_LEVEL=INFO
SSY_API_KEY=your-api-key
BIG_MODEL_NAME=selected-big-model
SMALL_MODEL_NAME=selected-small-model
```

您也可以通过环境变量覆盖这些设置。

## 🔧 开发

### 前置要求

- Go 1.21 或更高版本
- Make (可选)

### 开发命令

```bash
# 运行开发版本
make dev

# 运行测试
make test

# 格式化代码
make fmt

# 代码检查
make lint

# 构建所有平台
make build-all
```

### 项目结构

```
├── cmd/cli/            # CLI 应用程序
├── internal/
│   ├── cli/           # CLI 相关功能
│   ├── config/        # 配置管理
│   ├── handlers/      # HTTP 处理器
│   ├── middleware/    # 中间件
│   ├── models/        # 数据模型
│   ├── server/        # 服务器
│   └── services/      # 业务逻辑
├── build.sh           # 构建脚本 (Linux/macOS)
├── build.bat          # 构建脚本 (Windows)
├── Makefile           # Make 构建文件
└── main.go            # 主程序
```

## 🐛 故障排除

### 服务无法启动

1. 检查端口 3180 是否被占用
2. 确保 API 密钥有效
3. 查看配置是否正确: `claudeproxy config`

### 模型列表获取失败

1. 检查网络连接
2. 验证 API 密钥是否有效
3. 确保能访问 `https://router.shengsuanyun.com`

### 配置文件丢失

运行 `claudeproxy setup` 重新初始化配置。

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 支持

如果您遇到问题或有建议，请：

1. 查看 [Issues](https://github.com/your-repo/issues) 页面
2. 创建新的 Issue
3. 联系支持团队
