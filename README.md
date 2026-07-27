# Claude Code Proxy SSY

![License](https://img.shields.io/github/license/SSYCloud/claude-code-proxy-ssy) ![Release](https://img.shields.io/github/v/release/SSYCloud/claude-code-proxy-ssy)  ![Platform](https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-blue)  ![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)


本项目（Claude Code Proxy SSY）是一个命令行工具，可以将Claude API转换为胜算云格式，让您在Claude的应用程序中使用胜算云全球模型API。


## ✨ 功能特性

- 🚀 **简单易用**: 一键设置和启动
- 🔧 **交互式配置**: 引导式配置向导
- 🌐 **多平台支持**: 支持 Windows、macOS、Linux
- 🔄 **模型选择**: 支持选择不同的大小模型
- 📱 **后台运行**: 服务可在后台运行
- ⚙️ **配置管理**: 简单的配置修改和查看

## 📦 安装

### 前置条件
1. 注册 [胜算云](https://www.shengsuanyun.com) , 限时注册赠送免费额度
2. 安装Claude Code

```shell
npm install -g @anthropic-ai/claude-code
```

国内安装
```shell
npm config set registry https://registry.npmmirror.com
npm install -g @anthropic-ai/claude-code
```


### 方式一: 快速安装（推荐）

**Linux/macOS:**
```bash
sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_`uname -s`_`uname -m`
sudo chmod +x /usr/local/bin/claudeproxy
```
国内安装方式
```bash
sudo curl -o /usr/local/bin/claudeproxy -L https://shengsuanyun.oss-cn-shanghai.aliyuncs.com/claude-code-proxy-ssy/claudeproxy_`uname -s`_`uname -m`
sudo chmod +x /usr/local/bin/claudeproxy
```

**Windows:**

```bash
Invoke-WebRequest -Uri "https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_Windows_x86_64.exe" -OutFile "claudeproxy.exe"
```

国内安装方式：

```bash
Invoke-WebRequest -Uri "https://shengsuanyun.oss-cn-shanghai.aliyuncs.com/claude-code-proxy-ssy/claudeproxy_Windows_x86_64.exe" -OutFile "claudeproxy.exe"
```

### 方式二: 手动下载

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

### 方式三: 从源码构建

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
- 保存配置到 `~/.claudeproxy/config.json`

### 2. 启动服务

```bash
claudeproxy start
```

服务将在后台启动，默认监听 `http://0.0.0.0:3180`

**自动配置 Claude 环境变量**: 服务启动成功后，会自动设置以下环境变量，方便Claude Desktop等应用直接使用：

- `ANTHROPIC_BASE_URL=http://0.0.0.0:3180` (或您配置的HOST:PORT)
- `ANTHROPIC_AUTH_TOKEN=claudeproxy`

### 3. 使用服务

现在您可以将任何支持 OpenAI API 的应用程序配置为使用 `http://0.0.0.0:3180` 作为 API 端点。

对于Claude Desktop等原生支持Anthropic API的应用，环境变量已自动配置，无需额外设置。

## 📋 命令使用

### 1. 基本命令

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

# 无代理模式运行 Claude Code
claudeproxy code
```

### 2. 配置修改

使用 `claudeproxy set` 命令可以:

- 修改 API 密钥
- 重新选择模型
- 查看当前配置
- 重新初始化配置

### 3. 清理配置

使用 `claudeproxy clean` 命令可以完全清除所有项目相关的配置：

- 停止正在运行的服务
- 清除所有环境变量（包括ANTHROPIC_*变量，当前终端和全局环境）
- 删除配置文件
- 需要重启终端以确保环境变量完全清除

## ⚙️ 配置选项

默认配置保存在 `~/.claudeproxy/config.json` 文件中:

```json
{
  "ssy_api_key": "**********",
  "big_model_name": "****",
  "small_model_name": "****",
  "base_url": "https://router.shengsuanyun.com/api/v1",
  "referrer_url": "https://www.shengsuanyun.com",
  "app_name": "ClaudeCodeProxy",
  "app_version": "0.1.4",
  "host": "0.0.0.0",
  "port": "3180",
  "reload": "true",
  "open_claude_cache": "true",
  "log_level": "INFO"
}
```

您也可以通过环境变量覆盖这些设置。

## ⚙️ 使用claude code

```bash
claude
```

## 🐛 故障排除

### 1. 服务无法启动

1. 检查端口 3180 是否被占用
2. 确保 API 密钥有效
3. 查看配置是否正确: `claudeproxy config`

### 2. 模型列表获取失败

1. 检查网络连接
2. 验证 API 密钥是否有效
3. 确保能访问 `https://router.shengsuanyun.com`

### 3. 配置文件丢失

运行 `claudeproxy setup` 重新初始化配置。

### 4. 全局代理冲突问题

如果你设置了全局HTTP代理（如在 `~/.zshrc` 或 `~/.bashrc` 中设置了 `http_proxy`, `https_proxy`, `all_proxy` 等），Claude Code 可能会通过代理尝试访问本地服务器，导致连接失败。

**推荐解决方法：**

- **使用内置无代理命令** (最简单)

   直接使用内置的无代理模式命令启动 Claude Code：
   
   ```bash
   # macOS/Linux:
   claudeproxy code
   
   # Windows:
   claudeproxy.exe code
   ```
   
   此命令会自动禁用代理设置并启动 Claude Code，无需额外配置。支持所有平台（Windows, macOS, Linux）。

**其他解决方法：创建专用启动脚本***

   
   创建一个不使用代理的启动脚本 `run_claude.sh`：
   
   ```bash
   #!/bin/bash
   
   # 临时取消代理设置
   unset http_proxy https_proxy all_proxy HTTP_PROXY HTTPS_PROXY ALL_PROXY
   
   # 确保设置NO_PROXY
   export NO_PROXY="localhost,127.0.0.1,0.0.0.0"
   export no_proxy="localhost,127.0.0.1,0.0.0.0"
   
   # 运行Claude Code
   claude "$@"
   ```
   
   添加执行权限并使用：
   
   ```bash
   chmod +x run_claude.sh
   ./run_claude.sh
   ```


### 5. 网络问题排查（Windows电脑常见异常）

- 在新终端测试不同的访问地址

    ```bash
    curl -v http://127.0.0.1:3180/health
    curl -v http://localhost:3180/health  
    curl -v http://0.0.0.0:3180/health
    ```
- 选择可以访问通的Host，并手动修改`~/.claudeproxy/config.json` 文件中的

    ```json
    "host": "能访问通的Host",
    ```

- 停止服务后重新启动

    ```bash
    claudeproxy stop
    claudeproxy start
    ```

- 开启新的终端使用claude

### 6. 日志排查

```bash
claudeproxy log -l 100
```

查看是否有 `/v1/messages` 请求，如果没有请排查本地网络问题：

- 是否设置全局 HTTP_PROXY
- 是否有本地安全软件阻止3180端口访问
   
如果有 `/v1/messages` 请求，但是有报错，请提交 [Issues](https://github.com/SSYCloud/claude-code-proxy-ssy/issues)


## 🔧 开发

### 1. 前置要求

- Go 1.21 或更高版本
- Make (可选)

### 2. 开发命令

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

### 3. 项目结构

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

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📞 支持

如果您遇到问题或有建议，请：

1. 查看 [Issues](https://github.com/your-repo/issues) 页面
2. 创建新的 Issue
3. 联系支持团队
