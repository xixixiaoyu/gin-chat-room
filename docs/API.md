# API 文档

## 基础信息

- **Base URL**: `http://localhost:8080/api/v1`
- **认证方式**: JWT Bearer Token
- **Content-Type**: `application/json`

## 认证接口

### 用户注册

**POST** `/auth/register`

注册新用户账号。

**请求体**:
```json
{
  "username": "string",     // 用户名，3-50字符，必填
  "email": "string",        // 邮箱地址，必填
  "password": "string",     // 密码，最少6位，必填
  "nickname": "string"      // 昵称，可选
}
```

**响应**:
```json
{
  "token": "string",        // JWT token
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "测试用户",
    "avatar": "",
    "is_online": false,
    "last_seen": null
  }
}
```

**错误响应**:
```json
{
  "error": "Username or email already exists"
}
```

### 用户登录

**POST** `/auth/login`

用户登录获取访问令牌。

**请求体**:
```json
{
  "username": "string",     // 用户名或邮箱，必填
  "password": "string"      // 密码，必填
}
```

**响应**:
```json
{
  "token": "string",        // JWT token
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "测试用户",
    "avatar": "",
    "is_online": true,
    "last_seen": null
  }
}
```

## 用户接口

### 获取用户资料

**GET** `/profile`

获取当前登录用户的资料信息。

**请求头**:
```
Authorization: Bearer <token>
```

**响应**:
```json
{
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "测试用户",
    "avatar": "",
    "is_online": true,
    "last_seen": null
  }
}
```

### 更新用户资料

**PUT** `/profile`

更新当前用户的资料信息。

**请求头**:
```
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "nickname": "string",     // 新昵称，可选
  "avatar": "string"        // 头像URL，可选
}
```

**响应**:
```json
{
  "user": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "新昵称",
    "avatar": "avatar_url",
    "is_online": true,
    "last_seen": null
  }
}
```

## 房间接口

### 获取房间列表

**GET** `/rooms`

获取用户可见的房间列表。

**请求头**:
```
Authorization: Bearer <token>
```

**查询参数**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认20，最大100
- `search`: 搜索关键词，可选

**响应**:
```json
{
  "rooms": [
    {
      "id": 1,
      "name": "大厅",
      "description": "欢迎来到聊天室大厅！",
      "is_private": false,
      "max_members": 1000,
      "creator_id": 1,
      "member_count": 5,
      "created_at": "2023-01-01T00:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 1,
    "total_pages": 1
  }
}
```

### 创建房间

**POST** `/rooms`

创建新的聊天室。

**请求头**:
```
Authorization: Bearer <token>
```

**请求体**:
```json
{
  "name": "string",         // 房间名称，必填，1-100字符
  "description": "string",  // 房间描述，可选，最多500字符
  "is_private": false,      // 是否私有房间，默认false
  "password": "string",     // 私有房间密码，私有房间时可选
  "max_members": 100        // 最大成员数，默认100
}
```

**响应**:
```json
{
  "room": {
    "id": 2,
    "name": "新房间",
    "description": "房间描述",
    "is_private": false,
    "max_members": 100,
    "creator_id": 1,
    "member_count": 1,
    "created_at": "2023-01-01T00:00:00Z"
  }
}
```

### 获取房间详情

**GET** `/rooms/{id}`

获取指定房间的详细信息。

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `id`: 房间ID

**响应**:
```json
{
  "room": {
    "id": 1,
    "name": "大厅",
    "description": "欢迎来到聊天室大厅！",
    "is_private": false,
    "max_members": 1000,
    "creator_id": 1,
    "member_count": 5,
    "created_at": "2023-01-01T00:00:00Z"
  }
}
```

### 加入房间

**POST** `/rooms/{id}/join`

加入指定的聊天室。

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `id`: 房间ID

**请求体**:
```json
{
  "password": "string"      // 私有房间密码，私有房间时必填
}
```

**响应**:
```json
{
  "message": "Successfully joined room"
}
```

### 离开房间

**POST** `/rooms/{id}/leave`

离开指定的聊天室。

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `id`: 房间ID

**响应**:
```json
{
  "message": "Successfully left room"
}
```

### 获取房间消息

**GET** `/rooms/{id}/messages`

获取房间的历史消息。

**请求头**:
```
Authorization: Bearer <token>
```

**路径参数**:
- `id`: 房间ID

**查询参数**:
- `page`: 页码，默认1
- `page_size`: 每页数量，默认50，最大100

**响应**:
```json
{
  "messages": [
    {
      "id": 1,
      "room_id": 1,
      "user_id": 1,
      "type": "text",
      "content": "Hello, World!",
      "created_at": "2023-01-01T00:00:00Z",
      "user": {
        "id": 1,
        "username": "testuser",
        "nickname": "测试用户",
        "avatar": ""
      }
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 50,
    "total": 1,
    "total_pages": 1
  }
}
```

## WebSocket 接口

### 连接 WebSocket

**GET** `/ws?room_id={room_id}`

建立 WebSocket 连接进行实时通信。

**请求头**:
```
Authorization: Bearer <token>
Upgrade: websocket
Connection: Upgrade
```

**查询参数**:
- `room_id`: 房间ID

### WebSocket 消息格式

#### 发送消息
```json
{
  "type": "message",
  "room_id": 1,
  "content": "Hello, World!"
}
```

#### 接收消息
```json
{
  "type": "message",
  "room_id": 1,
  "data": {
    "id": 1,
    "room_id": 1,
    "user_id": 1,
    "type": "text",
    "content": "Hello, World!",
    "created_at": "2023-01-01T00:00:00Z",
    "user": {
      "id": 1,
      "username": "testuser",
      "nickname": "测试用户",
      "avatar": ""
    }
  }
}
```

#### 用户加入通知
```json
{
  "type": "user_joined",
  "room_id": 1,
  "data": {
    "user": {
      "id": 2,
      "username": "newuser",
      "nickname": "新用户",
      "avatar": ""
    }
  }
}
```

#### 用户离开通知
```json
{
  "type": "user_left",
  "room_id": 1,
  "data": {
    "user_id": 2
  }
}
```

#### 在线用户列表
```json
{
  "type": "online_users",
  "room_id": 1,
  "data": {
    "users": [
      {
        "id": 1,
        "username": "testuser",
        "nickname": "测试用户",
        "avatar": ""
      }
    ]
  }
}
```

## 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权 |
| 403 | 禁止访问 |
| 404 | 资源不存在 |
| 409 | 资源冲突 |
| 500 | 服务器内部错误 |

## 示例代码

### JavaScript 客户端示例

```javascript
// 登录
const loginResponse = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    username: 'testuser',
    password: 'password123'
  })
});

const { token } = await loginResponse.json();

// 建立 WebSocket 连接
const ws = new WebSocket(`ws://localhost:8080/api/v1/ws?room_id=1`);

ws.onopen = () => {
  console.log('WebSocket connected');
};

ws.onmessage = (event) => {
  const message = JSON.parse(event.data);
  console.log('Received:', message);
};

// 发送消息
ws.send(JSON.stringify({
  type: 'message',
  room_id: 1,
  content: 'Hello, World!'
}));
```
