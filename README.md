# 实时聊天室项目

一个基于 Go + Gin + WebSocket + SQLite/PostgreSQL + Redis 的现代化实时聊天室应用。

## 🚀 功能特性

### 核心功能
- ✅ 用户注册/登录系统（JWT 认证）
- ✅ 多个聊天室支持
- ✅ 实时消息发送和接收
- ✅ 在线用户列表显示
- ✅ 聊天记录持久化存储
- ✅ 响应式前端设计
- ✅ 私有房间支持（密码保护）
- ✅ 房间成员管理

### 技术特性
- 🔐 JWT 身份验证
- 🔄 WebSocket 实时通信
- 💾 数据库持久化（SQLite/PostgreSQL）
- 🚀 Redis 缓存和消息队列（可选）
- 📱 移动端适配
- 🎨 现代化 UI 设计

## 🛠 技术栈

### 后端
- **框架**: Gin (Go Web 框架)
- **数据库**: SQLite (默认) / PostgreSQL
- **ORM**: GORM
- **缓存**: Redis (可选)
- **WebSocket**: Gorilla WebSocket
- **认证**: JWT
- **密码加密**: bcrypt

### 前端
- **基础**: HTML5 + CSS3 + JavaScript (ES6+)
- **样式**: 响应式设计，支持移动端
- **图标**: Font Awesome
- **WebSocket**: 原生 WebSocket API

## 📦 项目结构

```
gin-chat-room/
├── cmd/                    # 应用程序入口
│   └── main.go
├── config/                 # 配置管理
│   └── config.go
├── internal/               # 内部包
│   ├── auth/              # JWT 认证
│   ├── database/          # 数据库初始化
│   ├── handlers/          # HTTP 处理器
│   ├── middleware/        # 中间件
│   ├── models/            # 数据模型
│   ├── services/          # 业务服务
│   └── websocket/         # WebSocket 处理
├── pkg/                   # 公共包
│   ├── logger/            # 日志工具
│   └── utils/             # 工具函数
├── web/                   # 前端资源
│   ├── static/            # 静态文件
│   │   ├── css/
│   │   ├── js/
│   │   └── images/
│   └── templates/         # HTML 模板
├── tests/                 # 测试文件
├── scripts/               # 脚本文件
├── docs/                  # 文档
├── .env                   # 环境变量
├── .env.example           # 环境变量示例
├── go.mod                 # Go 模块
├── go.sum                 # Go 依赖
└── README.md              # 项目说明
```

## 🚀 快速开始

### 环境要求
- Go 1.19+
- Redis (可选，用于缓存和消息队列)
- PostgreSQL (可选，默认使用 SQLite)

### 安装步骤

1. **克隆项目**
```bash
git clone <repository-url>
cd gin-chat-room
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置环境变量**
```bash
cp .env.example .env
# 编辑 .env 文件，配置数据库和其他设置
```

4. **启动应用**
```bash
go run cmd/main.go
```

5. **访问应用**
打开浏览器访问: http://localhost:8080

### 环境变量配置

```env
# 服务器配置
SERVER_PORT=8080
GIN_MODE=debug

# 数据库配置
DB_TYPE=sqlite                    # sqlite 或 postgres
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=
DB_PASSWORD=
DB_DATABASE=chatroom.db
DB_SSLMODE=disable

# Redis 配置（可选）
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT 配置
JWT_SECRET=your-very-secret-key-change-this-in-production
JWT_EXPIRE_TIME=24
```

## 📖 API 文档

### 认证接口

#### 用户注册
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123",
  "nickname": "测试用户"
}
```

#### 用户登录
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}
```

### 房间接口

#### 获取房间列表
```http
GET /api/v1/rooms
Authorization: Bearer <token>
```

#### 创建房间
```http
POST /api/v1/rooms
Authorization: Bearer <token>
Content-Type: application/json

{
  "name": "新房间",
  "description": "房间描述",
  "is_private": false,
  "max_members": 100
}
```

#### 加入房间
```http
POST /api/v1/rooms/{id}/join
Authorization: Bearer <token>
```

### WebSocket 连接

```javascript
const ws = new WebSocket('ws://localhost:8080/api/v1/ws?room_id=1');

// 发送消息
ws.send(JSON.stringify({
  type: 'message',
  room_id: 1,
  content: 'Hello, World!'
}));
```

## 🧪 测试

运行单元测试:
```bash
go test ./tests/... -v
```

运行特定测试:
```bash
go test ./tests/auth_test.go -v
```

## 🐳 Docker 部署

### 使用 Docker Compose

1. **创建 docker-compose.yml**
```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_TYPE=postgres
      - DB_HOST=postgres
      - DB_USERNAME=chatroom
      - DB_PASSWORD=password
      - DB_DATABASE=chatroom
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: chatroom
      POSTGRES_USER: chatroom
      POSTGRES_PASSWORD: password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

2. **启动服务**
```bash
docker-compose up -d
```

## 🔧 开发指南

### 添加新功能

1. **添加数据模型** - 在 `internal/models/` 中定义
2. **创建处理器** - 在 `internal/handlers/` 中实现
3. **添加路由** - 在 `cmd/main.go` 中注册
4. **编写测试** - 在 `tests/` 中添加测试用例

### 代码规范

- 遵循 Go 官方代码规范
- 使用 `gofmt` 格式化代码
- 添加必要的注释，特别是公共函数
- 编写单元测试

## 🤝 贡献指南

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 打开 Pull Request

## 📝 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 🙏 致谢

- [Gin](https://github.com/gin-gonic/gin) - Go Web 框架
- [GORM](https://gorm.io/) - Go ORM 库
- [Gorilla WebSocket](https://github.com/gorilla/websocket) - WebSocket 实现
- [Font Awesome](https://fontawesome.com/) - 图标库

## 📞 联系方式

如有问题或建议，请提交 Issue 或联系开发者。
