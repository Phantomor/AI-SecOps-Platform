<template>
  <div class="terminal-page-container">
    <div class="header">
      <div class="back-button" @click="goBack">[ ESC ] ABORT</div>
      <div class="title-container">
        <h1 class="title">Compliance Copilot <span class="accent-text">:: 合规审计</span></h1>
        <span class="cursor">_</span>
      </div>
      <div class="chat-id">SESSION_ID: {{ chatId }}</div>
    </div>
    
    <div class="content-wrapper">
      <div class="chat-area">
        <ChatRoom 
          :messages="messages" 
          :connection-status="connectionStatus"
          ai-type="super"
          @send-message="sendMessage"
          @stop-generating="handleStopGenerating" 
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useHead } from '@vueuse/head'
import ChatRoom from '../components/ChatRoom.vue'
import { chatWithComplianceCopilot } from '../api'

useHead({
  title: 'Compliance Copilot | 合规审计 - ZeroTrust Sentinel',
  meta: [
    { name: 'description', content: 'DevSecOps 合规顾问，提供安全编码审查、漏洞修复建议与合规性核查。' }
  ]
})

const router = useRouter()
const messages = ref([])
const chatId = ref('')
const connectionStatus = ref('disconnected')
let eventSource = null

const handleStopGenerating = () => {
  if (eventSource && typeof eventSource.close === 'function') {
    eventSource.close()
  }
  connectionStatus.value = 'disconnected'
  
  if (messages.value.length > 0) {
    const lastMsgIndex = messages.value.length - 1
    if (!messages.value[lastMsgIndex].isUser) {
      messages.value[lastMsgIndex].content += '\n\n`[WARN] PROCESS_TERMINATED_BY_USER`'
    }
  }
}

const generateChatId = () => {
  return 'COMP_' + Math.random().toString(36).substring(2, 10).toUpperCase()
}

const addMessage = (content, isUser) => {
  messages.value.push({
    content,
    isUser,
    time: new Date().getTime()
  })
}

const sendMessage = async (message) => {
  addMessage(message, true)
  
  if (eventSource) {
    eventSource.close()
  }
  
  const aiMessageIndex = messages.value.length
  addMessage('', false)
  connectionStatus.value = 'connecting'
  
  const onMessage = (data) => {
    if (data === '[DONE]') {
      connectionStatus.value = 'disconnected'
      return
    }
    if (aiMessageIndex < messages.value.length) {
      messages.value[aiMessageIndex].content += data
    }
  }

  const onError = (error) => {
    console.error('SSE Error:', error)
    connectionStatus.value = 'error'
  }
  
  eventSource = await chatWithComplianceCopilot(message, chatId.value, onMessage, onError)
}

const goBack = () => {
  router.push('/')
}

onMounted(() => {
  chatId.value = generateChatId()
  addMessage('[SYSTEM BOOT] DevSecOps 合规顾问已上线。\n\n> 零信任安全模块加载完毕。\n> 实时审计引擎状态：[READY]\n\n请提交需要审计的代码片段、架构设计或告警日志...', false)
})

onBeforeUnmount(() => {
  if (eventSource) {
    eventSource.close()
  }
})
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
}

/* ================= 布局控制：确保全屏铺满 ================= */
.terminal-page-container {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 60px);
  position: fixed;
  width: 100%;
  top: 60px; /* 对应导航栏高度 */
  overflow: hidden;
  flex: 1;
  min-height: 0;      
  background-color: var(--bg-deep);
  font-family: 'Fira Code', 'Inter', monospace;
}
.header {
  flex-shrink: 0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background-color: var(--bg-card);
  border-bottom: 1px solid var(--border-color);
  z-index: 10;
}

.back-button {
  font-size: 13px;
  cursor: pointer;
  color: var(--text-dim);
  transition: all 0.2s;
}

.back-button:hover {
  color: var(--accent-blue);
  text-shadow: 0 0 8px rgba(88, 166, 255, 0.4);
}

.title-container {
  display: flex;
  align-items: center;
}

.title {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
  color: #f0f6fc;
}

.accent-text {
  color: var(--text-dim);
  font-weight: normal;
  font-size: 14px;
  margin-left: 8px;
}

.cursor {
  display: inline-block;
  width: 8px;
  height: 18px;
  background-color: var(--accent-blue);
  margin-left: 8px;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

.chat-id {
  font-size: 12px;
  color: var(--text-dim);
}

/* ================= 内容区排版 ================= */
.content-wrapper, .chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;      
}

/* ================= ChatRoom 样式深度穿透 ================= */
:deep(.chat-container) {
  border: none !important;
  background-color: transparent !important;
  border-radius: 0 !important;
}

/* 消息左右对齐加固 */
:deep(.message-wrapper) {
  display: flex !important;
  width: 100% !important;
}
:deep(.ai-message) { justify-content: flex-start !important; margin-right: auto !important; }
:deep(.user-message) { justify-content: flex-end !important; margin-left: auto !important; }

:deep(.chat-input-container) {
  flex-shrink: 0 !important;
}

:deep(.chat-messages) {
  flex: 1 !important;
  height: 0 !important;
  overflow-y: auto !important;
}

:deep(.prompt-prefix) { color: var(--accent-blue) !important; }
:deep(.input-box) { color: var(--text-main) !important; }
:deep(.send-button) { color: var(--accent-blue) !important; }

/* 气泡风格 */
:deep(.ai-message .message-bubble) { 
  background-color: rgba(88, 166, 255, 0.05) !important; 
  border: 1px solid var(--border-color) !important; 
  color: var(--text-main) !important;
  max-width: 100% !important;
  overflow-x: auto !important;
}

:deep(.user-message .message-bubble) { 
  border: 1px dashed var(--accent-blue) !important; 
  color: var(--accent-blue) !important;
  background-color: transparent !important;
}

@media (max-width: 768px) {
  .header { padding: 12px 16px; }
  .title { font-size: 14px; }
  .chat-id { display: none; }
}
</style>