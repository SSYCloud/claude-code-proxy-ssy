#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}===== 启动 Claude Code Provider Proxy 服务 =====${NC}"

# 检查环境变量文件是否存在
if [ ! -f .env ]; then
    echo -e "${YELLOW}环境变量文件 .env 不存在，将从 .env.example 创建${NC}"
    cp .env.example .env
    echo -e "${YELLOW}请编辑 .env 文件设置您的 API 密钥和其他配置${NC}"
    echo -e "${RED}按 Ctrl+C 退出并编辑 .env 文件，或按 Enter 继续${NC}"
    read -r
fi

# 检查虚拟环境
if [ ! -d "venv" ]; then
    echo -e "${YELLOW}未检测到虚拟环境，正在创建...${NC}"
    python -m venv venv
    echo -e "${GREEN}虚拟环境已创建${NC}"
fi

# 激活虚拟环境
echo -e "${YELLOW}激活虚拟环境...${NC}"
source venv/bin/activate

# 安装依赖
echo -e "${YELLOW}安装依赖...${NC}"
pip install -r requirements.txt

# 清除可能存在的环境变量
echo -e "${YELLOW}清除环境变量...${NC}"
unset OPENAI_API_KEY
unset BIG_MODEL_NAME
unset SMALL_MODEL_NAME
unset BASE_URL
unset REFERRER_URL
unset APP_NAME
unset APP_VERSION
unset LOG_LEVEL
unset LOG_FILE_PATH
unset HOST
unset PORT
unset RELOAD
unset OPEN_CLAUDE_CACHE

# 启动应用
echo -e "${GREEN}启动应用...${NC}"
cd src && python -m main

# 捕获退出码
EXIT_CODE=$?

# 如果应用异常退出，显示错误信息
if [ $EXIT_CODE -ne 0 ]; then
    echo -e "${RED}应用异常退出，退出码: $EXIT_CODE${NC}"
    echo -e "${YELLOW}请检查日志获取更多信息${NC}"
fi

# 退出虚拟环境
deactivate

exit $EXIT_CODE
