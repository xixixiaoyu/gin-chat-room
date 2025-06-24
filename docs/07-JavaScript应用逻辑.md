# 07 - JavaScript 应用逻辑

## 📋 概述

本章节将详细介绍聊天室应用的 JavaScript 应用逻辑实现，包括单页应用架构设计、WebSocket 客户端实现、状态管理和用户交互处理。

## 🎯 学习目标

- 掌握单页应用 (SPA) 的架构设计
- 学会实现 WebSocket 客户端和连接管理
- 理解前端状态管理和数据流
- 掌握用户交互和事件处理

## 🏗️ 应用架构设计

### 整体架构

```
ChatApp (主应用类)
├── 认证管理 (Authentication)
│   ├── 登录/注册
│   ├── Token 管理
│   └── 用户状态
├── 页面管理 (Page Management)
│   ├── 页面切换
│   ├── 路由管理
│   └── 状态同步
├── WebSocket 管理 (WebSocket Management)
│   ├── 连接建立
│   ├── 消息处理
│   ├── 重连机制
│   └── 心跳检测
├── 房间管理 (Room Management)
│   ├── 房间列表
│   ├── 房间操作
│   └── 成员管理
└── UI 管理 (UI Management)
    ├── 消息渲染
    ├── 用户界面
    └── 交互反馈
```

### 设计模式

1. **单例模式**: 确保应用只有一个实例
2. **观察者模式**: WebSocket 事件处理
3. **状态模式**: 页面状态管理
4. **策略模式**: 不同消息类型的处理

## 📱 主应用类实现

创建 `web/static/js/app.js`：

```javascript
// 聊天室应用主文件
class ChatApp {
    constructor() {
        // 应用状态
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || 'null');
        this.currentRoom = null;
        this.ws = null;
        this.rooms = [];
        this.onlineUsers = [];
        
        // 配置
        this.config = {
            apiBaseUrl: '/api/v1',
            wsBaseUrl: this.getWebSocketUrl(),
            reconnectInterval: 3000,
            heartbeatInterval: 30000,
            maxReconnectAttempts: 5
        };
        
        // 状态管理
        this.state = {
            isConnected: false,
            isReconnecting: false,
            reconnectAttempts: 0
        };
        
        this.init();
    }

    // 初始化应用
    init() {
        this.bindEvents();
        this.setupErrorHandling();
        
        // 检查是否已登录
        if (this.token && this.user) {
            this.showRoomsPage();
            this.loadRooms();
        } else {
            this.showLoginPage();
        }
    }

    // 获取 WebSocket URL
    getWebSocketUrl() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        return `${protocol}//${window.location.host}/api/v1/ws`;
    }

    // 绑定事件监听器
    bindEvents() {
        // 登录表单事件
        document.getElementById('login-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleLogin();
        });

        // 注册表单事件
        document.getElementById('register-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleRegister();
        });

        // 创建房间事件
        document.getElementById('create-room-btn').addEventListener('click', () => {
            this.showCreateRoomModal();
        });

        document.getElementById('create-room-form').addEventListener('submit', (e) => {
            e.preventDefault();
            this.handleCreateRoom();
        });

        // 退出登录
        document.getElementById('logout-btn').addEventListener('click', () => {
            this.handleLogout();
        });

        // 返回房间列表
        document.getElementById('back-to-rooms').addEventListener('click', () => {
            this.leaveCurrentRoom();
            this.showRoomsPage();
        });

        // 发送消息
        document.getElementById('send-btn').addEventListener('click', () => {
            this.sendMessage();
        });

        document.getElementById('message-input').addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                this.sendMessage();
            }
        });

        // 搜索房间
        document.getElementById('rooms-search').addEventListener('input', (e) => {
            this.searchRooms(e.target.value);
        });

        // 模态框事件
        document.querySelectorAll('.modal-close, .modal-cancel').forEach(btn => {
            btn.addEventListener('click', () => {
                this.hideModals();
            });
        });

        // 私有房间密码显示/隐藏
        document.getElementById('room-private').addEventListener('change', (e) => {
            const passwordGroup = document.getElementById('password-group');
            passwordGroup.style.display = e.target.checked ? 'block' : 'none';
        });

        // 点击模态框外部关闭
        document.addEventListener('click', (e) => {
            if (e.target.classList.contains('modal')) {
                this.hideModals();
            }
        });

        // 窗口关闭前清理
        window.addEventListener('beforeunload', () => {
            if (this.ws) {
                this.ws.close();
            }
        });

        // 网络状态监听
        window.addEventListener('online', () => {
            this.showToast('网络连接已恢复', 'success');
            if (this.currentRoom && !this.state.isConnected) {
                this.connectWebSocket();
            }
        });

        window.addEventListener('offline', () => {
            this.showToast('网络连接已断开', 'error');
        });
    }

    // 设置错误处理
    setupErrorHandling() {
        window.addEventListener('error', (e) => {
            console.error('Global error:', e.error);
            this.showToast('应用发生错误，请刷新页面', 'error');
        });

        window.addEventListener('unhandledrejection', (e) => {
            console.error('Unhandled promise rejection:', e.reason);
            this.showToast('网络请求失败，请重试', 'error');
        });
    }

    // 页面切换方法
    showLoginPage() {
        this.hideAllPages();
        document.getElementById('login-page').classList.add('active');
    }

    showRoomsPage() {
        this.hideAllPages();
        document.getElementById('rooms-page').classList.add('active');
    }

    showChatPage() {
        this.hideAllPages();
        document.getElementById('chat-page').classList.add('active');
    }

    hideAllPages() {
        document.querySelectorAll('.page').forEach(page => {
            page.classList.remove('active');
        });
    }
}

// 认证相关方法
ChatApp.prototype.handleLogin = async function() {
    const username = document.getElementById('login-username').value.trim();
    const password = document.getElementById('login-password').value;

    if (!username || !password) {
        this.showToast('请填写完整信息', 'error');
        return;
    }

    this.showLoading();

    try {
        const response = await this.apiRequest('/auth/login', {
            method: 'POST',
            body: JSON.stringify({ username, password }),
        });

        if (response.token) {
            this.token = response.token;
            this.user = response.user;
            
            localStorage.setItem('token', this.token);
            localStorage.setItem('user', JSON.stringify(this.user));
            
            this.showToast('登录成功', 'success');
            this.showRoomsPage();
            this.loadRooms();
        }
    } catch (error) {
        this.showToast(error.message || '登录失败', 'error');
    } finally {
        this.hideLoading();
    }
};

ChatApp.prototype.handleRegister = async function() {
    const username = document.getElementById('register-username').value.trim();
    const email = document.getElementById('register-email').value.trim();
    const nickname = document.getElementById('register-nickname').value.trim();
    const password = document.getElementById('register-password').value;

    if (!username || !email || !password) {
        this.showToast('请填写必填信息', 'error');
        return;
    }

    if (password.length < 6) {
        this.showToast('密码至少6位', 'error');
        return;
    }

    this.showLoading();

    try {
        const response = await this.apiRequest('/auth/register', {
            method: 'POST',
            body: JSON.stringify({ username, email, nickname, password }),
        });

        if (response.token) {
            this.token = response.token;
            this.user = response.user;
            
            localStorage.setItem('token', this.token);
            localStorage.setItem('user', JSON.stringify(this.user));
            
            this.showToast('注册成功', 'success');
            this.showRoomsPage();
            this.loadRooms();
        }
    } catch (error) {
        this.showToast(error.message || '注册失败', 'error');
    } finally {
        this.hideLoading();
    }
};

ChatApp.prototype.handleLogout = function() {
    if (this.ws) {
        this.ws.close();
        this.ws = null;
    }
    
    this.token = null;
    this.user = null;
    this.currentRoom = null;
    this.state.isConnected = false;
    
    localStorage.removeItem('token');
    localStorage.removeItem('user');
    
    this.showLoginPage();
    this.showToast('已退出登录', 'info');
};

// API 请求封装
ChatApp.prototype.apiRequest = async function(endpoint, options = {}) {
    const url = this.config.apiBaseUrl + endpoint;
    
    const defaultOptions = {
        headers: {
            'Content-Type': 'application/json',
        },
    };

    if (this.token) {
        defaultOptions.headers['Authorization'] = `Bearer ${this.token}`;
    }

    const finalOptions = {
        ...defaultOptions,
        ...options,
        headers: {
            ...defaultOptions.headers,
            ...options.headers,
        },
    };

    const response = await fetch(url, finalOptions);
    const data = await response.json();

    if (!response.ok) {
        throw new Error(data.error || `HTTP ${response.status}`);
    }

    return data;
};

// 房间相关方法
ChatApp.prototype.loadRooms = async function() {
    try {
        const data = await this.apiRequest('/rooms');
        this.rooms = data.rooms || [];
        this.renderRooms(this.rooms);
    } catch (error) {
        this.showToast('加载房间失败: ' + error.message, 'error');
    }
};

ChatApp.prototype.renderRooms = function(rooms) {
    const roomsList = document.getElementById('rooms-list');
    
    if (rooms.length === 0) {
        roomsList.innerHTML = '<div class="no-rooms">暂无房间</div>';
        return;
    }

    roomsList.innerHTML = rooms.map(room => `
        <div class="room-card" onclick="app.joinRoom(${room.id})">
            <h3>
                ${this.escapeHtml(room.name)}
                ${room.is_private ? '<i class="fas fa-lock room-private"></i>' : ''}
            </h3>
            <p>${this.escapeHtml(room.description || '暂无描述')}</p>
            <div class="room-meta">
                <span><i class="fas fa-users"></i> ${room.member_count}/${room.max_members}</span>
                <span><i class="fas fa-user"></i> ${room.creator_id === this.user.id ? '我创建' : '其他人创建'}</span>
            </div>
        </div>
    `).join('');
};

ChatApp.prototype.searchRooms = function(query) {
    if (!query.trim()) {
        this.renderRooms(this.rooms);
        return;
    }

    const filteredRooms = this.rooms.filter(room => 
        room.name.toLowerCase().includes(query.toLowerCase()) ||
        (room.description && room.description.toLowerCase().includes(query.toLowerCase()))
    );

    this.renderRooms(filteredRooms);
};

// 工具方法
ChatApp.prototype.escapeHtml = function(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
};

ChatApp.prototype.formatTime = function(dateString) {
    const date = new Date(dateString);
    return date.toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit'
    });
};

// 表单切换函数
function showLogin() {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.auth-form').forEach(form => form.classList.remove('active'));
    
    document.querySelector('.tab-btn').classList.add('active');
    document.getElementById('login-form').classList.add('active');
}

function showRegister() {
    document.querySelectorAll('.tab-btn').forEach(btn => btn.classList.remove('active'));
    document.querySelectorAll('.auth-form').forEach(form => form.classList.remove('active'));
    
    document.querySelectorAll('.tab-btn')[1].classList.add('active');
    document.getElementById('register-form').classList.add('active');
}

// 初始化应用
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new ChatApp();
});
```

### JavaScript 架构特点

1. **类式设计**: 使用 ES6 类组织代码结构
2. **模块化**: 功能按模块分离，便于维护
3. **错误处理**: 完善的错误捕获和用户反馈
4. **状态管理**: 统一的应用状态管理
5. **事件驱动**: 基于事件的交互处理

## 🔌 WebSocket 客户端实现

### WebSocket 连接管理

```javascript
// WebSocket 相关方法
ChatApp.prototype.connectWebSocket = function() {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        return;
    }

    const wsUrl = `${this.config.wsBaseUrl}?room_id=${this.currentRoom.id}`;
    
    try {
        this.ws = new WebSocket(wsUrl);
        this.setupWebSocketEvents();
    } catch (error) {
        console.error('WebSocket connection failed:', error);
        this.handleWebSocketError();
    }
};

ChatApp.prototype.setupWebSocketEvents = function() {
    this.ws.onopen = () => {
        console.log('WebSocket connected');
        this.state.isConnected = true;
        this.state.isReconnecting = false;
        this.state.reconnectAttempts = 0;
        
        this.showToast('连接成功', 'success');
        this.startHeartbeat();
    };

    this.ws.onmessage = (event) => {
        try {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        } catch (error) {
            console.error('Error parsing WebSocket message:', error);
        }
    };

    this.ws.onclose = (event) => {
        console.log('WebSocket disconnected:', event.code, event.reason);
        this.state.isConnected = false;
        this.stopHeartbeat();
        
        if (this.currentRoom && !this.state.isReconnecting) {
            this.handleWebSocketReconnect();
        }
    };

    this.ws.onerror = (error) => {
        console.error('WebSocket error:', error);
        this.handleWebSocketError();
    };
};

ChatApp.prototype.handleWebSocketMessage = function(message) {
    switch (message.type) {
        case 'message':
            this.addMessage(message.data);
            break;
        case 'user_joined':
            this.handleUserJoined(message.data.user);
            break;
        case 'user_left':
            this.handleUserLeft(message.data.user_id);
            break;
        case 'online_users':
            this.updateOnlineUsers(message.data.users);
            break;
        case 'error':
            this.showToast(message.data.message || '服务器错误', 'error');
            break;
        default:
            console.log('Unknown message type:', message.type);
    }
};

ChatApp.prototype.handleWebSocketReconnect = function() {
    if (this.state.reconnectAttempts >= this.config.maxReconnectAttempts) {
        this.showToast('连接失败，请刷新页面重试', 'error');
        return;
    }

    this.state.isReconnecting = true;
    this.state.reconnectAttempts++;
    
    this.showToast(`正在重连... (${this.state.reconnectAttempts}/${this.config.maxReconnectAttempts})`, 'info');
    
    setTimeout(() => {
        if (this.currentRoom) {
            this.connectWebSocket();
        }
    }, this.config.reconnectInterval);
};

ChatApp.prototype.handleWebSocketError = function() {
    this.state.isConnected = false;
    this.showToast('连接出现问题', 'error');
};

// 心跳检测
ChatApp.prototype.startHeartbeat = function() {
    this.heartbeatTimer = setInterval(() => {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ type: 'ping' }));
        }
    }, this.config.heartbeatInterval);
};

ChatApp.prototype.stopHeartbeat = function() {
    if (this.heartbeatTimer) {
        clearInterval(this.heartbeatTimer);
        this.heartbeatTimer = null;
    }
};

// 发送消息
ChatApp.prototype.sendMessage = function() {
    const input = document.getElementById('message-input');
    const content = input.value.trim();

    if (!content || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
        return;
    }

    const message = {
        type: 'message',
        room_id: this.currentRoom.id,
        content: content,
    };

    this.ws.send(JSON.stringify(message));
    input.value = '';
};
```

### 消息处理和渲染

```javascript
// 消息相关方法
ChatApp.prototype.loadMessages = async function() {
    try {
        const data = await this.apiRequest(`/rooms/${this.currentRoom.id}/messages`);
        const messagesContainer = document.getElementById('messages-container');
        messagesContainer.innerHTML = '';
        
        data.messages.forEach(message => {
            this.addMessage(message, false);
        });

        this.scrollToBottom();
    } catch (error) {
        console.error('Load messages error:', error);
        this.showToast('加载消息失败', 'error');
    }
};

ChatApp.prototype.addMessage = function(messageData, scroll = true) {
    const messagesContainer = document.getElementById('messages-container');
    const messageElement = document.createElement('div');
    
    const isOwn = messageData.user_id === this.user.id;
    const isSystem = messageData.type === 'system';
    
    messageElement.className = `message ${isOwn ? 'own' : ''} ${isSystem ? 'system' : ''}`;
    
    if (isSystem) {
        messageElement.innerHTML = `
            <div class="message-content">
                <div class="message-text">${this.escapeHtml(messageData.content)}</div>
            </div>
        `;
    } else {
        const avatar = messageData.user.nickname ? messageData.user.nickname.charAt(0).toUpperCase() : 'U';
        const time = this.formatTime(messageData.created_at);
        
        messageElement.innerHTML = `
            <div class="message-avatar">${avatar}</div>
            <div class="message-content">
                <div class="message-header">
                    <span class="message-author">${this.escapeHtml(messageData.user.nickname)}</span>
                    <span class="message-time">${time}</span>
                </div>
                <div class="message-text">${this.escapeHtml(messageData.content)}</div>
            </div>
        `;
    }
    
    messagesContainer.appendChild(messageElement);
    
    if (scroll) {
        this.scrollToBottom();
    }
};

ChatApp.prototype.scrollToBottom = function() {
    const messagesContainer = document.getElementById('messages-container');
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
};

// 在线用户管理
ChatApp.prototype.handleUserJoined = function(user) {
    this.showToast(`${user.nickname} 加入了房间`, 'info');
};

ChatApp.prototype.handleUserLeft = function(userId) {
    const user = this.onlineUsers.find(u => u.id === userId);
    if (user) {
        this.showToast(`${user.nickname} 离开了房间`, 'info');
    }
};

ChatApp.prototype.updateOnlineUsers = function(users) {
    this.onlineUsers = users;
    this.renderOnlineUsers();
    
    document.getElementById('online-count').textContent = `${users.length} 人在线`;
};

ChatApp.prototype.renderOnlineUsers = function() {
    const onlineUsersContainer = document.getElementById('online-users');
    
    onlineUsersContainer.innerHTML = this.onlineUsers.map(user => {
        const avatar = user.nickname ? user.nickname.charAt(0).toUpperCase() : 'U';
        return `
            <div class="user-item">
                <div class="user-avatar">${avatar}</div>
                <div class="user-info">
                    <div class="user-name">${this.escapeHtml(user.nickname)}</div>
                    <div class="user-status">在线</div>
                </div>
            </div>
        `;
    }).join('');
};
```

## 🎯 下一步

在下一章节中，我们将详细介绍测试和部署相关内容，包括：
- 单元测试编写
- 集成测试
- 性能优化
- 生产环境部署

通过本章节的学习，您应该已经掌握了：
- 单页应用的架构设计
- WebSocket 客户端的实现
- 前端状态管理
- 用户交互处理
