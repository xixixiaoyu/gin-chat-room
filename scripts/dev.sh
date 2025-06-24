#!/bin/bash

# 开发环境启动脚本

set -e

echo "🛠️  启动开发环境..."

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo "❌ Go 未安装，请先安装 Go 1.19+"
    exit 1
fi

# 检查环境变量文件
if [ ! -f .env ]; then
    echo "📝 创建环境变量文件..."
    cp .env.example .env
    echo "✅ 已创建 .env 文件，使用默认配置"
fi

# 安装依赖
echo "📦 安装依赖..."
go mod tidy

# 运行测试
echo "🧪 运行测试..."
go test ./tests/... -v

if [ $? -eq 0 ]; then
    echo "✅ 所有测试通过"
else
    echo "❌ 测试失败"
    exit 1
fi

# 启动应用
echo "🚀 启动应用..."
echo "🌐 访问地址: http://localhost:8080"
echo "📋 按 Ctrl+C 停止应用"
echo ""

go run cmd/main.go
