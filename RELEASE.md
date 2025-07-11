# Claude Code Proxy 发布指南

这个项目包含了完整的 GitHub Release 自动化流程，让用户可以轻松下载和安装二进制文件。

## 🚀 快速发布流程

### 方法 1: 使用脚本发布（推荐）

1. **构建二进制文件**：
   ```bash
   ./build-release.sh v0.1.0
   ```

2. **创建 GitHub Release**：
   ```bash
   ./create-release.sh v0.1.0
   ```

### 方法 2: 使用 GitHub Actions 自动发布

1. **推送标签到 GitHub**：
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

2. **GitHub Actions 会自动**：
   - 构建所有平台的二进制文件
   - 创建 GitHub Release
   - 上传所有二进制文件

## 📦 用户安装方式

### 1. 一键安装脚本
```bash
curl -fsSL https://raw.githubusercontent.com/SSYCloud/claude-code-proxy-ssy/main/install.sh | bash
```

### 2. 手动下载（类似 cog）
```bash
# Linux/macOS
sudo curl -o /usr/local/bin/claudeproxy -L https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_$(uname -s)_$(uname -m)
sudo chmod +x /usr/local/bin/claudeproxy

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_Windows_x86_64.exe" -OutFile "claudeproxy.exe"
```

## 📋 文件说明

- `build-release.sh` - 构建所有平台的二进制文件
- `create-release.sh` - 创建 GitHub Release 并上传文件
- `install.sh` - 用户安装脚本
- `.github/workflows/release.yml` - GitHub Actions 自动发布流程

## 🎯 二进制文件命名规则

我们遵循 GitHub Release 的标准命名规则：

- Linux x86_64: `claudeproxy_Linux_x86_64`
- Linux ARM64: `claudeproxy_Linux_arm64`
- macOS Intel: `claudeproxy_Darwin_x86_64`
- macOS Apple Silicon: `claudeproxy_Darwin_arm64`
- Windows x86_64: `claudeproxy_Windows_x86_64.exe`
- Windows ARM64: `claudeproxy_Windows_arm64.exe`

这样用户可以使用 `$(uname -s)` 和 `$(uname -m)` 来自动检测系统并下载对应的二进制文件。

## 🔧 前置要求

### 使用脚本发布
- 安装 [GitHub CLI](https://cli.github.com/)
- 登录 GitHub: `gh auth login`
- Go 1.21+ 用于构建

### 使用 GitHub Actions
- 只需要推送标签即可，无需本地环境

## 🎉 发布后的链接

发布完成后，用户可以通过以下链接访问：

- **Release 页面**: https://github.com/SSYCloud/claude-code-proxy-ssy/releases
- **最新版本**: https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest
- **特定文件**: https://github.com/SSYCloud/claude-code-proxy-ssy/releases/latest/download/claudeproxy_Linux_x86_64

## 📝 注意事项

1. 确保在发布前测试所有平台的二进制文件
2. 版本号使用语义化版本控制（如 v1.0.0）
3. 每次发布都会覆盖同名的文件
4. 建议在 README.md 中添加安装说明

## 🐛 故障排除

如果遇到问题：

1. 检查 GitHub CLI 是否已登录
2. 确认标签是否已推送到远程仓库
3. 检查构建目录是否包含所有必需的二进制文件
4. 验证 GitHub repository 的权限设置
