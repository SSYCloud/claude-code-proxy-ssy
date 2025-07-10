# Claude Code Provider Proxy (Go版本)

一个高性能的代理服务，将Anthropic Claude API请求转换为OpenAI兼容的API格式。使用Go语言和Gin框架重构，提供更好的性能和并发处理能力。

## 功能特性

### 核心功能
- ✅ 将Anthropic Claude API请求转换为OpenAI API格式
- ✅ 支持流式和非流式响应
- ✅ 完整的令牌计数和使用情况跟踪
- ✅ 全面的错误处理和结构化日志记录
- ✅ 工具使用支持（Function Calling）
- ✅ **Cache Control支持** - 完整支持Anthropic的缓存控制功能
- ✅ **消息顺序保证** - 确保转换前后消息顺序完全一致

### 高级功能
- ✅ 多种内容类型支持（文本、图片、工具调用、工具结果）
- ✅ 完整的流式响应处理
- ✅ 速率限制和并发控制
- ✅ CORS支持和安全头设置
- ✅ 健康检查和状态监控
- ✅ Docker容器化支持
- ✅ 结构化配置管理

### 内容类型支持
- **文本内容**: 完整支持带cache_control的文本消息
- **图片内容**: 支持base64编码的图片处理
- **工具使用**: 完整的Function Calling支持
- **工具结果**: 工具执行结果的双向转换
- **缓存控制**: 保持Anthropic cache_control设置

## 项目结构

```
claude-code-provider-proxy/
├── main.go                    # 应用入口点
├── internal/                  # 内部包
│   ├── config/               # 配置管理
│   │   └── config.go
│   ├── models/               # 数据模型
│   │   ├── anthropic.go      # Anthropic API模型（支持cache_control）
│   │   └── errors.go         # 错误处理模型
│   ├── services/             # 服务层
│   │   ├── conversion.go     # 转换服务（保证消息顺序）
│   │   ├── streaming.go      # 流式处理服务
│   │   ├── token_counting.go # 令牌计数服务
│   │   └── openai_client.go  # OpenAI客户端
│   ├── middleware/           # 中间件
│   │   └── middleware.go     # 认证、CORS、日志等
│   ├── handlers/             # HTTP处理器
│   │   └── handlers.go
│   └── server/               # 服务器配置
│       └── server.go
├── go.mod                    # Go模块定义
├── go.sum                    # 依赖锁定文件
├── .env.example              # 环境变量示例
├── Dockerfile                # Docker构建文件
├── docker-compose.yml        # Docker Compose配置
├── run_app.sh               # 应用启动脚本
└── run_docker.sh            # Docker启动脚本
```

## 安装和使用

### 方法1：直接运行

1. 确保安装了Go 1.21或更高版本
2. 克隆仓库并进入目录：
```bash
git clone <repository-url>
cd claude-code-provider-proxy
```

3. 配置环境变量：
```bash
cp .env.example .env
# 编辑.env文件，设置你的OpenAI API密钥
```

4. 运行应用：
```bash
./run_app.sh
```

### 方法2：使用Docker

1. 配置环境变量：
```bash
cp .env.example .env
# 编辑.env文件
```

2. 使用Docker运行：
```bash
./run_docker.sh
```

## API端点

### 核心端点
- `GET /` - 健康检查
- `GET /health` - 健康检查
- `GET /status` - 详细状态信息
- `POST /v1/messages` - 创建消息（完全兼容Anthropic API）
- `POST /v1/messages/count_tokens` - 计算令牌数量

### 工具端点
- `GET /v1/models` - 获取可用模型列表
- `POST /v1/validate` - 验证API密钥

## 使用示例

### 基本消息请求
```bash
curl -X POST http://localhost:8000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1000,
    "messages": [
      {
        "role": "user",
        "content": "Hello, how are you?"
      }
    ],
    "system": "You are a helpful assistant."
  }'
```

### 带Cache Control的请求
```bash
curl -X POST http://localhost:8000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1000,
    "messages": [
      {
        "role": "user",
        "content": [
          {
            "type": "text",
            "text": "This is cached content",
            "cache_control": {"type": "ephemeral"}
          }
        ]
      }
    ]
  }'
```

### 流式请求
```bash
curl -X POST http://localhost:8000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-3-opus-20240229",
    "max_tokens": 1000,
    "stream": true,
    "messages": [
      {
        "role": "user",
        "content": "Tell me a story"
      }
    ]
  }'
```

## 配置选项

### 环境变量
```bash
# 服务器配置
PORT=8000                    # 服务端口
HOST=0.0.0.0                # 绑定地址

# OpenAI API配置
OPENAI_API_KEY=your-key      # OpenAI API密钥（必需）
OPENAI_BASE_URL=https://api.openai.com/v1  # OpenAI API基础URL
OPENAI_MODEL=gpt-4           # 默认使用的OpenAI模型
OPENAI_MAX_TOKENS=4096       # 最大令牌数

# 日志配置
LOG_LEVEL=info               # 日志级别 (debug, info, warn, error)

# 速率限制
RATE_LIMIT=100               # 每分钟请求限制
```

## 特性详解

### Cache Control支持
- 完整保持Anthropic的`cache_control`设置
- 在转换过程中标记缓存控制的内容
- 响应时恢复原始的缓存控制信息

### 消息顺序保证
- 转换前后消息顺序完全一致
- 内置验证机制确保顺序正确性
- 支持复杂的消息结构和嵌套内容

### 内容类型处理
- **文本**: 支持简单文本和复杂文本结构
- **图片**: 处理base64编码的图片数据
- **工具使用**: 完整的Function Calling支持
- **工具结果**: 双向转换工具执行结果

### 错误处理
- 结构化错误响应
- 详细的错误日志记录
- 优雅的错误恢复机制

## 性能特性

- **高并发**: 基于Go的高性能并发处理
- **低延迟**: 优化的请求转换和处理流程
- **内存效率**: 流式处理减少内存占用
- **可扩展**: 支持水平扩展和负载均衡

## 监控和日志

### 健康检查
```bash
curl http://localhost:8000/health
```

### 详细状态
```bash
curl http://localhost:8000/status
```

### 日志格式
使用结构化JSON日志，包含：
- 请求ID跟踪
- 性能指标
- 错误详情
- Cache control使用情况

## 开发和贡献

### 本地开发
```bash
# 安装依赖
go mod download

# 运行测试
go test ./...

# 构建
go build -o claude-proxy .

# 运行
./claude-proxy
```

### Docker开发
```bash
# 构建镜像
docker build -t claude-proxy .

# 运行容器
docker run -p 8000:8000 --env-file .env claude-proxy
```

## 许可证

[MIT](LICENSE)

## 更新日志

### v1.0.0 (Go重构版本)
- ✅ 完整的Python功能移植
- ✅ Cache Control完整支持
- ✅ 消息顺序严格保证
- ✅ 高性能Go实现
- ✅ 完整的错误处理
- ✅ Docker容器化支持
- ✅ 结构化日志和监控
