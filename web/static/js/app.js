// 聊天室应用主文件
class ChatApp {
  constructor() {
    this.token = localStorage.getItem('token')
    this.user = JSON.parse(localStorage.getItem('user') || 'null')
    this.currentRoom = null
    this.ws = null
    this.rooms = []
    this.onlineUsers = []

    this.init()
  }

  init() {
    this.bindEvents()

    // 检查是否已登录
    if (this.token && this.user) {
      this.showRoomsPage()
      this.loadRooms()
    } else {
      this.showLoginPage()
    }
  }

  bindEvents() {
    // 登录表单事件
    document.getElementById('login-form').addEventListener('submit', (e) => {
      e.preventDefault()
      this.handleLogin()
    })

    // 注册表单事件
    document.getElementById('register-form').addEventListener('submit', (e) => {
      e.preventDefault()
      this.handleRegister()
    })

    // 创建房间事件
    document.getElementById('create-room-btn').addEventListener('click', () => {
      this.showCreateRoomModal()
    })

    document.getElementById('create-room-form').addEventListener('submit', (e) => {
      e.preventDefault()
      this.handleCreateRoom()
    })

    // 退出登录
    document.getElementById('logout-btn').addEventListener('click', () => {
      this.handleLogout()
    })

    // 返回房间列表
    document.getElementById('back-to-rooms').addEventListener('click', () => {
      this.leaveCurrentRoom()
      this.showRoomsPage()
    })

    // 发送消息
    document.getElementById('send-btn').addEventListener('click', () => {
      this.sendMessage()
    })

    document.getElementById('message-input').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') {
        this.sendMessage()
      }
    })

    // 搜索房间
    document.getElementById('rooms-search').addEventListener('input', (e) => {
      this.searchRooms(e.target.value)
    })

    // 模态框事件
    document.querySelectorAll('.modal-close, .modal-cancel').forEach((btn) => {
      btn.addEventListener('click', () => {
        this.hideModals()
      })
    })

    // 私有房间密码显示/隐藏
    document.getElementById('room-private').addEventListener('change', (e) => {
      const passwordGroup = document.getElementById('password-group')
      passwordGroup.style.display = e.target.checked ? 'block' : 'none'
    })

    // 点击模态框外部关闭
    document.addEventListener('click', (e) => {
      if (e.target.classList.contains('modal')) {
        this.hideModals()
      }
    })
  }

  // 页面切换
  showLoginPage() {
    this.hideAllPages()
    document.getElementById('login-page').classList.add('active')
  }

  showRoomsPage() {
    this.hideAllPages()
    document.getElementById('rooms-page').classList.add('active')
  }

  showChatPage() {
    this.hideAllPages()
    document.getElementById('chat-page').classList.add('active')
  }

  hideAllPages() {
    document.querySelectorAll('.page').forEach((page) => {
      page.classList.remove('active')
    })
  }

  // 认证相关方法
  async handleLogin() {
    const username = document.getElementById('login-username').value.trim()
    const password = document.getElementById('login-password').value

    if (!username || !password) {
      this.showToast('请填写完整信息', 'error')
      return
    }

    this.showLoading()

    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, password }),
      })

      const data = await response.json()

      if (response.ok) {
        this.token = data.token
        this.user = data.user

        localStorage.setItem('token', this.token)
        localStorage.setItem('user', JSON.stringify(this.user))

        this.showToast('登录成功', 'success')
        this.showRoomsPage()
        this.loadRooms()
      } else {
        this.showToast(data.error || '登录失败', 'error')
      }
    } catch (error) {
      console.error('Login error:', error)
      this.showToast('网络错误，请重试', 'error')
    } finally {
      this.hideLoading()
    }
  }

  async handleRegister() {
    const username = document.getElementById('register-username').value.trim()
    const email = document.getElementById('register-email').value.trim()
    const nickname = document.getElementById('register-nickname').value.trim()
    const password = document.getElementById('register-password').value

    if (!username || !email || !password) {
      this.showToast('请填写必填信息', 'error')
      return
    }

    if (password.length < 6) {
      this.showToast('密码至少6位', 'error')
      return
    }

    this.showLoading()

    try {
      const response = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ username, email, nickname, password }),
      })

      const data = await response.json()

      if (response.ok) {
        this.token = data.token
        this.user = data.user

        localStorage.setItem('token', this.token)
        localStorage.setItem('user', JSON.stringify(this.user))

        this.showToast('注册成功', 'success')
        this.showRoomsPage()
        this.loadRooms()
      } else {
        this.showToast(data.error || '注册失败', 'error')
      }
    } catch (error) {
      console.error('Register error:', error)
      this.showToast('网络错误，请重试', 'error')
    } finally {
      this.hideLoading()
    }
  }

  handleLogout() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }

    this.token = null
    this.user = null
    this.currentRoom = null

    localStorage.removeItem('token')
    localStorage.removeItem('user')

    this.showLoginPage()
    this.showToast('已退出登录', 'info')
  }

  // 房间相关方法
  async loadRooms() {
    try {
      const response = await fetch('/api/v1/rooms', {
        headers: {
          Authorization: `Bearer ${this.token}`,
        },
      })

      const data = await response.json()

      if (response.ok) {
        this.rooms = data.rooms || []
        this.renderRooms(this.rooms)
      } else {
        this.showToast(data.error || '加载房间失败', 'error')
      }
    } catch (error) {
      console.error('Load rooms error:', error)
      this.showToast('网络错误', 'error')
    }
  }

  renderRooms(rooms) {
    const roomsList = document.getElementById('rooms-list')

    if (rooms.length === 0) {
      roomsList.innerHTML = '<div class="no-rooms">暂无房间</div>'
      return
    }

    roomsList.innerHTML = rooms
      .map(
        (room) => `
            <div class="room-card" onclick="app.joinRoom(${room.id})">
                <h3>
                    ${room.name}
                    ${room.is_private ? '<i class="fas fa-lock room-private"></i>' : ''}
                </h3>
                <p>${room.description || '暂无描述'}</p>
                <div class="room-meta">
                    <span><i class="fas fa-users"></i> ${room.member_count}/${
          room.max_members
        }</span>
                    <span><i class="fas fa-user"></i> ${
                      room.creator_id === this.user.id ? '我创建' : '其他人创建'
                    }</span>
                </div>
            </div>
        `
      )
      .join('')
  }

  searchRooms(query) {
    if (!query.trim()) {
      this.renderRooms(this.rooms)
      return
    }

    const filteredRooms = this.rooms.filter(
      (room) =>
        room.name.toLowerCase().includes(query.toLowerCase()) ||
        (room.description && room.description.toLowerCase().includes(query.toLowerCase()))
    )

    this.renderRooms(filteredRooms)
  }

  async joinRoom(roomId) {
    this.showLoading()

    try {
      // 先尝试加入房间
      const response = await fetch(`/api/v1/rooms/${roomId}/join`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      })

      if (response.ok || response.status === 409) {
        // 409 表示已经是成员
        // 获取房间信息
        const roomResponse = await fetch(`/api/v1/rooms/${roomId}`, {
          headers: {
            Authorization: `Bearer ${this.token}`,
          },
        })

        const roomData = await roomResponse.json()

        if (roomResponse.ok) {
          this.currentRoom = roomData.room
          this.showChatPage()
          this.connectWebSocket()
          this.loadMessages()

          document.getElementById('current-room-name').textContent = this.currentRoom.name
        } else {
          this.showToast(roomData.error || '获取房间信息失败', 'error')
        }
      } else {
        const data = await response.json()
        this.showToast(data.error || '加入房间失败', 'error')
      }
    } catch (error) {
      console.error('Join room error:', error)
      this.showToast('网络错误', 'error')
    } finally {
      this.hideLoading()
    }
  }

  async handleCreateRoom() {
    const name = document.getElementById('room-name').value.trim()
    const description = document.getElementById('room-description').value.trim()
    const isPrivate = document.getElementById('room-private').checked
    const password = document.getElementById('room-password').value
    const maxMembers = parseInt(document.getElementById('room-max-members').value)

    if (!name) {
      this.showToast('请输入房间名称', 'error')
      return
    }

    this.showLoading()

    try {
      const response = await fetch('/api/v1/rooms', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${this.token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name,
          description,
          is_private: isPrivate,
          password,
          max_members: maxMembers,
        }),
      })

      const data = await response.json()

      if (response.ok) {
        this.showToast('房间创建成功', 'success')
        this.hideModals()
        this.loadRooms()

        // 重置表单
        document.getElementById('create-room-form').reset()
        document.getElementById('password-group').style.display = 'none'
      } else {
        this.showToast(data.error || '创建房间失败', 'error')
      }
    } catch (error) {
      console.error('Create room error:', error)
      this.showToast('网络错误', 'error')
    } finally {
      this.hideLoading()
    }
  }

  leaveCurrentRoom() {
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.currentRoom = null
    this.onlineUsers = []
  }

  // WebSocket 相关方法
  connectWebSocket() {
    if (this.ws) {
      this.ws.close()
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws?room_id=${this.currentRoom.id}`

    this.ws = new WebSocket(wsUrl)
    this.ws.onopen = () => {
      console.log('WebSocket connected')
    }

    this.ws.onmessage = (event) => {
      const message = JSON.parse(event.data)
      this.handleWebSocketMessage(message)
    }

    this.ws.onclose = () => {
      console.log('WebSocket disconnected')
      // 尝试重连
      if (this.currentRoom) {
        setTimeout(() => {
          this.connectWebSocket()
        }, 3000)
      }
    }

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }
  }

  handleWebSocketMessage(message) {
    switch (message.type) {
      case 'message':
        this.addMessage(message.data)
        break
      case 'user_joined':
        this.handleUserJoined(message.data.user)
        break
      case 'user_left':
        this.handleUserLeft(message.data.user_id)
        break
      case 'online_users':
        this.updateOnlineUsers(message.data.users)
        break
    }
  }

  sendMessage() {
    const input = document.getElementById('message-input')
    const content = input.value.trim()

    if (!content || !this.ws || this.ws.readyState !== WebSocket.OPEN) {
      return
    }

    const message = {
      type: 'message',
      room_id: this.currentRoom.id,
      content: content,
    }

    this.ws.send(JSON.stringify(message))
    input.value = ''
  }

  async loadMessages() {
    try {
      const response = await fetch(`/api/v1/rooms/${this.currentRoom.id}/messages`, {
        headers: {
          Authorization: `Bearer ${this.token}`,
        },
      })

      const data = await response.json()

      if (response.ok) {
        const messagesContainer = document.getElementById('messages-container')
        messagesContainer.innerHTML = ''

        data.messages.forEach((message) => {
          this.addMessage(message, false)
        })

        this.scrollToBottom()
      }
    } catch (error) {
      console.error('Load messages error:', error)
    }
  }

  addMessage(messageData, scroll = true) {
    const messagesContainer = document.getElementById('messages-container')
    const messageElement = document.createElement('div')

    const isOwn = messageData.user_id === this.user.id
    const isSystem = messageData.type === 'system'

    messageElement.className = `message ${isOwn ? 'own' : ''} ${isSystem ? 'system' : ''}`

    if (isSystem) {
      messageElement.innerHTML = `
                <div class="message-content">
                    <div class="message-text">${messageData.content}</div>
                </div>
            `
    } else {
      const avatar = messageData.user.nickname
        ? messageData.user.nickname.charAt(0).toUpperCase()
        : 'U'
      const time = new Date(messageData.created_at).toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      })

      messageElement.innerHTML = `
                <div class="message-avatar">${avatar}</div>
                <div class="message-content">
                    <div class="message-header">
                        <span class="message-author">${messageData.user.nickname}</span>
                        <span class="message-time">${time}</span>
                    </div>
                    <div class="message-text">${messageData.content}</div>
                </div>
            `
    }

    messagesContainer.appendChild(messageElement)

    if (scroll) {
      this.scrollToBottom()
    }
  }

  scrollToBottom() {
    const messagesContainer = document.getElementById('messages-container')
    messagesContainer.scrollTop = messagesContainer.scrollHeight
  }

  // 在线用户相关方法
  handleUserJoined(user) {
    this.showToast(`${user.nickname} 加入了房间`, 'info')
    // 在线用户列表会通过 online_users 消息更新
  }

  handleUserLeft(userId) {
    const user = this.onlineUsers.find((u) => u.id === userId)
    if (user) {
      this.showToast(`${user.nickname} 离开了房间`, 'info')
    }
    // 在线用户列表会通过 online_users 消息更新
  }

  updateOnlineUsers(users) {
    this.onlineUsers = users
    this.renderOnlineUsers()

    document.getElementById('online-count').textContent = `${users.length} 人在线`
  }

  renderOnlineUsers() {
    const onlineUsersContainer = document.getElementById('online-users')

    onlineUsersContainer.innerHTML = this.onlineUsers
      .map((user) => {
        const avatar = user.nickname ? user.nickname.charAt(0).toUpperCase() : 'U'
        return `
                <div class="user-item">
                    <div class="user-avatar">${avatar}</div>
                    <div class="user-info">
                        <div class="user-name">${user.nickname}</div>
                        <div class="user-status">在线</div>
                    </div>
                </div>
            `
      })
      .join('')
  }

  // UI 辅助方法
  showCreateRoomModal() {
    document.getElementById('create-room-modal').classList.add('active')
  }

  hideModals() {
    document.querySelectorAll('.modal').forEach((modal) => {
      modal.classList.remove('active')
    })
  }

  showLoading() {
    document.getElementById('loading').classList.add('active')
  }

  hideLoading() {
    document.getElementById('loading').classList.remove('active')
  }

  showToast(message, type = 'info') {
    const toast = document.getElementById('toast')
    toast.textContent = message
    toast.className = `toast ${type} show`

    setTimeout(() => {
      toast.classList.remove('show')
    }, 3000)
  }
}

// 表单切换函数
function showLogin() {
  document.querySelectorAll('.tab-btn').forEach((btn) => btn.classList.remove('active'))
  document.querySelectorAll('.auth-form').forEach((form) => form.classList.remove('active'))

  document.querySelector('.tab-btn').classList.add('active')
  document.getElementById('login-form').classList.add('active')
}

function showRegister() {
  document.querySelectorAll('.tab-btn').forEach((btn) => btn.classList.remove('active'))
  document.querySelectorAll('.auth-form').forEach((form) => form.classList.remove('active'))

  document.querySelectorAll('.tab-btn')[1].classList.add('active')
  document.getElementById('register-form').classList.add('active')
}

// 初始化应用
let app
document.addEventListener('DOMContentLoaded', () => {
  app = new ChatApp()
})
