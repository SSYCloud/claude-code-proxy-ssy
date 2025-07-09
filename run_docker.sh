#!/bin/bash

# 设置颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}===== 使用 Docker 启动 Claude Code Provider Proxy 服务 =====${NC}"

# 检查环境变量文件是否存在
if [ ! -f .env ]; then
    echo -e "${YELLOW}环境变量文件 .env 不存在，将从 .env.example 创建${NC}"
    cp .env.example .env
    echo -e "${YELLOW}请编辑 .env 文件设置您的 API 密钥和其他配置${NC}"
    echo -e "${RED}按 Ctrl+C 退出并编辑 .env 文件，或按 Enter 继续${NC}"
    read -r
fi

# 检查 Docker 是否安装
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Docker 未安装，请先安装 Docker${NC}"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}Docker Compose 未安装，请先安装 Docker Compose${NC}"
    exit 1
fi

# 构建并启动容器
echo -e "${YELLOW}构建并启动 Docker 容器...${NC}"
docker-compose up --build -d

# 检查容器是否成功启动
if [ $? -eq 0 ]; then
    echo -e "${GREEN}Docker 容器已成功启动${NC}"
    echo -e "${GREEN}服务运行在 http://localhost:8000${NC}"
    echo -e "${YELLOW}使用以下命令查看日志:${NC}"
    echo -e "${YELLOW}docker-compose logs -f${NC}"
    echo -e "${YELLOW}使用以下命令停止服务:${NC}"
    echo -e "${YELLOW}docker-compose down${NC}"
else
    echo -e "${RED}Docker 容器启动失败${NC}"
    echo -e "${YELLOW}使用以下命令查看错误日志:${NC}"
    echo -e "${YELLOW}docker-compose logs${NC}"
    exit 1
fi
