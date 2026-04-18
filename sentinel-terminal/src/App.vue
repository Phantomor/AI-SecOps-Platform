<script setup>
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from './store'
import { userLogout } from './api'

const router = useRouter()
const userStore = useUserStore()

// 计算属性，检查用户是否登录
const isLoggedIn = computed(() => userStore.isLoggedIn)
const userAccount = computed(() => userStore.userInfo?.userAccount || '')

// 跳转到登录页面
const goToLogin = () => {
  router.push('/login')
}

// 处理退出登录
const handleLogout = async () => {
  try {
    await userLogout()
    userStore.clearUserInfo()
    userStore.saveToLocalStorage()
    // 退出登录后跳转到首页
    router.push('/')
  } catch (error) {
    console.error('退出登录失败:', error)
  }
}
</script>

<template>
  <div id="app">
    <nav class="top-nav">
      <div class="nav-content">
        <div class="nav-title">
          <router-link to="/" class="app-logo">
            <span class="logo-prefix">root@</span>
            <span class="logo-text">ZeroTrust_Sentinel</span>
            <span class="cursor">_</span>
          </router-link>
        </div>
        
        <div class="user-menu">
          <button v-if="!isLoggedIn" class="login-button" @click="goToLogin">
            [ AUTH_LOGIN ]
          </button>
          
          <div v-else class="user-dropdown">
            <button class="user-button">
              <span class="status-dot"></span>
              <span class="user-name">{{ userAccount }}</span>
              <span class="dropdown-icon">▼</span>
            </button>
            <div class="dropdown-menu">
              <button class="dropdown-item" @click="handleLogout">
                > DISCONNECT
              </button>
            </div>
          </div>
        </div>
      </div>
    </nav>
    
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style>
/* 引入全局等宽字体 */
@import url('https://fonts.googleapis.com/css2?family=Fira+Code:wght@400;600&family=Inter:wght@400;500;600&display=swap');

:root {
  --bg-deep: #0a0c10;      /* 全局深色背景 */
  --bg-card: #161b22;      /* 面板背景 */
  --accent-blue: #58a6ff;  /* 极光蓝强调色 */
  --text-main: #c9d1d9;    /* 主文本色 */
  --text-dim: #8b949e;     /* 辅助文本色 */
  --border-color: #30363d; /* 边框色 */
  --danger-color: #f85149; /* 警告色 */
}

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body {
  font-family: 'Fira Code', 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  font-size: 16px;
  color: var(--text-main);
  background-color: var(--bg-deep); /* 替换掉原来的亮色背景 */
  width: 100%;
  height: 100%;
  overflow-x: hidden;
  -webkit-font-smoothing: antialiased;
}

#app {
  width: 100%;
  min-height: 100vh; 
  display: flex;
  flex-direction: column;
}

/* ================= 顶部导航栏 ================= */
.top-nav {
  background-color: rgba(10, 12, 16, 0.85);
  border-bottom: 1px dashed var(--border-color); /* 使用虚线增强科技感 */
  position: sticky;
  top: 0;
  z-index: 1000;
  backdrop-filter: blur(8px);
}

.nav-content {
  max-width: 1400px;
  margin: 0 auto;
  padding: 0 20px;
  height: 60px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.nav-title .app-logo {
  display: flex;
  align-items: center;
  text-decoration: none;
}

.logo-prefix {
  color: var(--text-dim);
  font-size: 16px;
  margin-right: 2px;
}

.logo-text {
  font-size: 18px;
  font-weight: 600;
  color: #f0f6fc;
  letter-spacing: 0.5px;
}

.cursor {
  display: inline-block;
  width: 8px;
  height: 18px;
  background-color: var(--accent-blue);
  margin-left: 5px;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ================= 用户菜单 ================= */
.user-menu {
  position: relative;
}

.login-button {
  background: transparent;
  color: var(--accent-blue);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 6px 16px;
  font-size: 14px;
  font-family: inherit;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.2s ease;
}

.login-button:hover {
  border-color: var(--accent-blue);
  background-color: rgba(88, 166, 255, 0.1);
  box-shadow: 0 0 10px rgba(88, 166, 255, 0.2);
}

/* 用户下拉菜单 */
.user-dropdown {
  position: relative;
}

.user-button {
  display: flex;
  align-items: center;
  gap: 8px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--text-main);
  padding: 6px 12px;
  border-radius: 4px;
  cursor: pointer;
  font-family: inherit;
  font-size: 14px;
  transition: all 0.2s;
}

.status-dot {
  width: 8px;
  height: 8px;
  background-color: #2ea043; /* 在线绿点 */
  border-radius: 50%;
  box-shadow: 0 0 5px #2ea043;
}

.user-button:hover {
  background-color: var(--bg-card);
  border-color: var(--border-color);
}

.dropdown-icon {
  font-size: 10px;
  color: var(--text-dim);
  transition: transform 0.3s;
}

.user-dropdown:hover .dropdown-icon {
  transform: rotate(180deg);
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 8px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.5);
  min-width: 140px;
  display: none;
  z-index: 1001;
}

.user-dropdown:hover .dropdown-menu {
  display: block;
}

.dropdown-item {
  display: block;
  width: 100%;
  padding: 12px 16px;
  background: none;
  border: none;
  color: var(--danger-color); /* 退出按钮使用红色警示 */
  text-align: left;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  transition: background-color 0.2s;
}

.dropdown-item:hover {
  background-color: rgba(248, 81, 73, 0.1);
}

/* ================= 主内容区域 ================= */
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0; 
}

/* ================= 响应式 ================= */
@media (max-width: 768px) {
  html, body {
  font-family: 'Fira Code', 'Inter', sans-serif;
  font-size: 16px;
  color: var(--text-main);
  background-color: var(--bg-deep);
  width: 100%;
  height: 100%;
  margin: 0;
  overflow: hidden; 
  -webkit-font-smoothing: antialiased;
  }
  .nav-content {
    padding: 0 15px;
  }
  .logo-text {
    font-size: 16px;
  }
  .logo-prefix {
    display: none; /* 手机端隐藏 root@ 前缀节省空间 */
  }
}

/* ================= 暗黑版滚动条 ================= */
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: var(--bg-deep);
}

::-webkit-scrollbar-thumb {
  background: var(--border-color);
  border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
  background: var(--text-dim);
}
</style>