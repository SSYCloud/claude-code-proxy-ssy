# Claude Code Provider Proxy

一个代理服务，将Anthropic API请求路由到OpenAI兼容的API。

## 项目结构

```
claude-code-provider-proxy/
├── docs/                      # 文档
│   ├── cover.png
│   ├── example.png
│   └── mapping.md
├── src/                       # 源代码
│   ├── api/                   # API相关代码
│   │   ├── middleware.py      # 中间件
│   │   └── routes.py          # 路由定义
│   ├── config/                # 配置相关代码
│   │   └── settings.py        # 应用设置
│   ├── models/                # 数据模型
│   │   ├── anthropic.py       # Anthropic API模型
│   │   └── errors.py          # 错误处理模型
│   ├── services/              # 服务层
│   │   ├── conversion.py      # 转换服务
│   │   ├── streaming.py       # 流式处理服务
│   │   └── token_counting.py  # 令牌计数服务
│   ├── utils/                 # 工具函数
│   │   └── logging.py         # 日志工具
│   └── main.py                # 应用入口点
├── .env                       # 环境变量（不包含在版本控制中）
├── .env.example               # 环境变量示例
├── .gitignore                 # Git忽略文件
├── Dockerfile                 # Docker构建文件
├── docker-compose.yml         # Docker Compose配置
├── LICENSE                    # 许可证
├── pyproject.toml             # 项目配置
├── pytest.ini                 # Pytest配置
├── README.md                  # 项目说明
├── requirements.txt           # 依赖项
├── run_app.bat                # Windows应用启动脚本
├── run_app.sh                 # Linux/macOS应用启动脚本
├── run_docker.bat             # Windows Docker启动脚本
└── run_docker.sh              # Linux/macOS Docker启动脚本
```

## 功能特性

- 将Anthropic Claude API请求转换为OpenAI API格式
- 支持流式和非流式响应
- 令牌计数和使用情况跟踪
- 错误处理和日志记录
- 工具使用支持

## 安装

1. 克隆仓库：

```bash
git clone https://github.com/yourusername/claude-code-provider-proxy.git
cd claude-code-provider-proxy
```

2. 创建并激活虚拟环境：

```bash
python -m venv venv
source venv/bin/activate  # 在Windows上使用: venv\Scripts\activate
```

3. 安装依赖：

```bash
pip install -r requirements.txt
```

4. 配置环境变量：

```bash
cp .env.example .env
```

编辑`.env`文件，设置您的OpenAI API密钥和其他配置。

## 使用方法

### 方法1：直接运行

使用提供的脚本启动服务器：

**Linux/macOS:**
```bash
./run_app.sh
```

**Windows:**
```bash
run_app.bat
```

或者手动启动：

```bash
python -m src.main
```

### 方法2：使用Docker

使用提供的脚本通过Docker启动服务器：

**Linux/macOS:**
```bash
./run_docker.sh
```

**Windows:**
```bash
run_docker.bat
```

或者手动使用Docker Compose：

```bash
docker-compose up -d
```

服务器将在`http://localhost:8000`上运行（或根据您的环境变量配置的端口）。

### API端点

- `GET /` - 健康检查
- `POST /v1/messages` - 创建消息（与Anthropic API兼容）
- `POST /v1/messages/count_tokens` - 计算令牌数量

### 示例请求

```bash
curl -X POST http://localhost:8000/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: your-api-key" \
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

## 许可证

[MIT](LICENSE)
