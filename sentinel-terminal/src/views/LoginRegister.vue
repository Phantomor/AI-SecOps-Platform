<template>
  <div class="login-register-container">
    <div class="bg-overlay"></div>

    <div class="form-wrapper fade-in">
      <button class="back-button" @click="goHome">
        [ ESC ] ABORT
      </button>

      <div class="form-header">
        <div class="platform-logo">
          <span class="logo-prefix">root@</span>
          <span class="platform-title">ZeroTrust_Sentinel</span>
        </div>
        <h1>{{ isLogin ? 'SYSTEM_LOGIN' : 'REGISTER_NODE' }}<span class="cursor">_</span></h1>
        <p class="subtitle">
          {{ isLogin ? 'Please authenticate to access SecOps platform.' : 'Request access to deploy a new node.' }}
        </p>
      </div>
      
      <form @submit.prevent="handleSubmit" class="form">
        <div class="form-group">
          <span class="prompt-prefix">USR ></span>
          <input
            v-model="formData.userAccount"
            type="text"
            placeholder="ENTER_ACCOUNT_ID"
            required
            autocomplete="off"
          />
        </div>
        
        <div class="form-group">
          <span class="prompt-prefix">PWD ></span>
          <input
            v-model="formData.userPassword"
            type="password"
            placeholder="ENTER_SECURE_KEY"
            required
          />
        </div>
        
        <div v-if="!isLogin" class="form-group">
          <span class="prompt-prefix">CHK ></span>
          <input
            v-model="formData.checkPassword"
            type="password"
            placeholder="VERIFY_SECURE_KEY"
            required
          />
        </div>
        
        <div v-if="errorMessage" class="error-message">
          [ ERR ] {{ errorMessage }}
        </div>
        
        <button type="submit" class="submit-button" :disabled="isSubmitting">
          {{ isSubmitting ? 'PROCESSING...' : (isLogin ? '[ EXECUTE_LOGIN ]' : '[ INITIALIZE_NODE ]') }}
        </button>
        
        <div class="register-link">
          <span>{{ isLogin ? 'UNAUTHORIZED?' : 'ALREADY_HAVE_ACCESS?' }}</span>
          <button type="button" class="toggle-button" @click="toggleForm">
            {{ isLogin ? 'REQUEST_ACCESS' : 'RETURN_TO_LOGIN' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '../store'
import { userRegister, userLogin } from '../api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const route = useRoute()

// 返回首页
const goHome = () => {
  router.push('/')
}

// 监听路由 query.msg，若有则弹出 element-plus 警告
if (route.query.msg) {
  ElMessage.warning({ message: route.query.msg, customClass: 'cyber-toast' })
}

const userStore = useUserStore()

// 表单状态
const isLogin = ref(true) // true表示登录表单，false表示注册表单
const isSubmitting = ref(false)
const errorMessage = ref('')

// 表单数据
const formData = ref({
  userAccount: '',
  userPassword: '',
  checkPassword: ''
})

// 切换登录/注册表单
const toggleForm = () => {
  isLogin.value = !isLogin.value
  errorMessage.value = ''
  // 清空表单数据
  formData.value = {
    userAccount: '',
    userPassword: '',
    checkPassword: ''
  }
}

// 处理表单提交
const handleSubmit = async () => {
  // 清除之前的错误信息
  errorMessage.value = ''
  
  // 表单验证
  if (!formData.value.userAccount.trim()) {
    errorMessage.value = 'IDENTITY_REQUIRED: 请输入账号'
    return
  }
  
  if (!formData.value.userPassword.trim()) {
    errorMessage.value = 'KEY_REQUIRED: 请输入密码'
    return
  }
  
  // 注册表单需要验证两次密码是否一致
  if (!isLogin.value && formData.value.userPassword !== formData.value.checkPassword) {
    errorMessage.value = 'KEY_MISMATCH: 两次输入的密码不一致'
    return
  }
  
  isSubmitting.value = true
  
  try {
    if (isLogin.value) {
      // 登录
      const userInfo = await userLogin(formData.value.userAccount, formData.value.userPassword)
      // 保存用户信息到store
      userStore.setUserInfo(userInfo)
      userStore.saveToLocalStorage()
      ElMessage.success('AUTH_SUCCESS: 身份验证通过')
      // 跳转到首页
      router.push('/')
    } else {
      // 注册
      await userRegister(
        formData.value.userAccount,
        formData.value.userPassword,
        formData.value.checkPassword
      )
      ElMessage.success('NODE_CREATED: 节点注册成功，请重新验证')
      // 注册成功后切换到登录表单
      toggleForm()
    }
  } catch (error) {
    errorMessage.value = error.message || 'CONNECTION_FAILED: 操作失败，请重试'
    ElMessage.error(errorMessage.value)
    console.error('登录/注册失败:', error)
  } finally {
    isSubmitting.value = false
  }
}
</script>

<style scoped>
/* 继承全局暗黑变量 */
:root {
  --bg-deep: #0a0c10;
  --bg-card: #161b22;
  --accent-blue: #58a6ff;
  --text-main: #c9d1d9;
  --text-dim: #8b949e;
  --border-color: #30363d;
  --error-color: #f85149;
}

.login-register-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 100vh;
  background-color: var(--bg-deep); /* 完美融入全站背景 */
  position: relative;
  font-family: 'Fira Code', 'Inter', monospace;
  overflow: hidden;
}

/* 背景虚化层 */
.bg-overlay {
  position: absolute;
  top: 0; left: 0; width: 100%; height: 100%;
  background: radial-gradient(circle at center, rgba(88, 166, 255, 0.03) 0%, transparent 70%);
  pointer-events: none;
}

.form-wrapper {
  background: var(--bg-card);
  border-radius: 8px;
  padding: 45px 40px 35px;
  width: 100%;
  max-width: 420px;
  position: relative;
  box-shadow: 0 15px 35px rgba(0, 0, 0, 0.6);
  border: 1px solid var(--border-color);
  z-index: 2;
}

.fade-in {
  animation: slideUp 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes slideUp {
  from { opacity: 0; transform: translateY(20px); }
  to { opacity: 1; transform: translateY(0); }
}

/* 左上角返回按钮 */
.back-button {
  position: absolute;
  top: 15px;
  left: 15px;
  background: transparent;
  border: none;
  font-size: 12px;
  color: var(--text-dim);
  cursor: pointer;
  z-index: 10;
  font-family: inherit;
  transition: all 0.2s;
}

.back-button:hover {
  color: var(--accent-blue);
  text-shadow: 0 0 8px rgba(88, 166, 255, 0.4);
}

/* 头部信息 */
.form-header {
  text-align: left;
  margin-bottom: 35px;
  border-bottom: 1px dashed var(--border-color);
  padding-bottom: 20px;
}

.platform-logo {
  display: flex;
  align-items: center;
  margin-bottom: 10px;
  font-size: 14px;
}

.logo-prefix {
  color: var(--text-dim);
}

.platform-title {
  color: var(--accent-blue);
  font-weight: 600;
}

.form-header h1 {
  color: #f0f6fc;
  font-size: 24px;
  margin: 0 0 5px 0;
  font-weight: 600;
  letter-spacing: 1px;
}

.subtitle {
  font-size: 12px;
  color: var(--text-dim);
  margin: 0;
}

.cursor {
  display: inline-block;
  width: 10px;
  height: 22px;
  background-color: var(--accent-blue);
  vertical-align: bottom;
  margin-left: 5px;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* 表单输入区 */
.form {
  display: flex;
  flex-direction: column;
}

.form-group {
  margin-bottom: 20px;
  position: relative;
  display: flex;
  align-items: center;
  background: rgba(10, 12, 16, 0.5);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 0 15px;
  transition: all 0.3s;
}

.form-group:focus-within {
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 1px rgba(88, 166, 255, 0.2) inset;
}

.prompt-prefix {
  color: var(--accent-blue);
  font-size: 13px;
  font-weight: bold;
  margin-right: 12px;
  user-select: none;
}

.form-group input {
  flex: 1;
  padding: 14px 0;
  background: transparent;
  border: none;
  color: var(--text-main);
  font-size: 14px;
  font-family: inherit;
  outline: none;
}

.form-group input::placeholder {
  color: #484f58;
}

/* 错误提示框 */
.error-message {
  color: var(--error-color);
  font-size: 12px;
  margin-bottom: 20px;
  padding: 10px;
  background-color: rgba(248, 81, 73, 0.1);
  border-left: 3px solid var(--error-color);
  font-family: inherit;
}

/* 提交按钮 */
.submit-button {
  background: transparent;
  color: var(--accent-blue);
  border: 1px solid var(--accent-blue);
  border-radius: 4px;
  padding: 14px 20px;
  font-size: 15px;
  font-weight: 600;
  font-family: inherit;
  cursor: pointer;
  margin-top: 10px;
  transition: all 0.2s;
  letter-spacing: 1px;
}

.submit-button:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.1);
  box-shadow: 0 0 15px rgba(88, 166, 255, 0.2);
}

.submit-button:disabled {
  opacity: 0.4;
  border-color: var(--border-color);
  color: var(--text-dim);
  cursor: not-allowed;
}

/* 底部切换链接 */
.register-link {
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: 25px;
  color: var(--text-dim);
  font-size: 12px;
}

.toggle-button {
  background: none;
  border: none;
  color: var(--accent-blue);
  cursor: pointer;
  font-size: 12px;
  margin-left: 8px;
  padding: 0;
  font-family: inherit;
  font-weight: 600;
  text-decoration: underline;
  text-underline-offset: 4px;
  transition: color 0.2s;
}

.toggle-button:hover {
  color: #79c0ff;
}

/* 响应式适配 */
@media (max-width: 768px) {
  .form-wrapper {
    padding: 30px 20px;
    margin: 15px;
    border-radius: 6px;
  }
  .form-header h1 {
    font-size: 20px;
  }
}

/* ================= 拦截 Chrome 自动填充的默认样式 ================= */
.form-group input:-webkit-autofill,
.form-group input:-webkit-autofill:hover, 
.form-group input:-webkit-autofill:focus, 
.form-group input:-webkit-autofill:active {
  /* 利用 5000 秒的过渡延迟，让 Chrome 的蓝底永远无法渲染出来，保持我们的透明底色 */
  transition: background-color 5000s ease-in-out 0s;
  
  /* 强制把自动填充的文字颜色改回极客主题的主文本色 */
  -webkit-text-fill-color: var(--text-main) !important;
  
  /* 确保输入光标依然是极光蓝 */
  caret-color: var(--accent-blue) !important;
}
</style>