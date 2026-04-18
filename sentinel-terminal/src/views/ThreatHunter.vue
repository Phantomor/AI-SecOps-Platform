<template>
  <div class="terminal-page-container">
    <div class="header">
      <div class="back-button" @click="goBack">[ ESC ] ABORT</div>
      <div class="title-container">
        <h1 class="title">ThreatHunter <span class="accent-text">:: 威胁狩猎专家</span></h1>
        <span class="cursor">_</span>
      </div>
      <div class="chat-id">TRACE_ID: {{ chatId }}</div>
    </div>
    
    <div class="content-wrapper">
      <div class="chat-area">
        <ChatRoom 
          :messages="messages" 
          :connection-status="connectionStatus"
          ai-type="secops"
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
import { chatWithThreatHunter } from '../api'

useHead({
  title: 'ThreatHunter - ZeroTrust Sentinel',
  meta: [
    { name: 'description', content: 'ZeroTrust Sentinel 核心组件：具备 MCP 工具挂载与 RAG 检索能力的自动化安全研判平台。' }
  ]
})

const router = useRouter()
const messages = ref([])
const chatId = ref('')
const connectionStatus = ref('disconnected')
let eventSource = null

// 中断响应
const handleStopGenerating = () => {
  if (eventSource && typeof eventSource.close === 'function') {
    eventSource.close();
  }
  connectionStatus.value = 'disconnected';
  
  if (messages.value.length > 0) {
    const lastMsgIndex = messages.value.length - 1;
    if (!messages.value[lastMsgIndex].isUser) {
      messages.value[lastMsgIndex].content += '\n\n`[WARN] SESSION_TERMINATED_BY_USER`';
    }
  }
}

// 生成安全的会话 ID
const generateChatId = () => {
  return 'REQ_' + Math.random().toString(16).substring(2, 10).toUpperCase()
}

// 追加消息
const addMessage = (content, isUser) => {
  messages.value.push({
    content,
    isUser,
    time: new Date().getTime()
  })
}

// 发送消息与流式解析
const sendMessage = async (message) => {
  addMessage(message, true)
  
  if (eventSource && typeof eventSource.close === 'function') {
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
    // 拦截后端的思考标记，替换为硬核日志样式
    let formattedData = data;
    if (formattedData.includes('⚙️ 正在调用插件')) {
      formattedData = formattedData.replace('⚙️ 正在调用插件', '`[AUDIT LOG] Injecting module`') 
                                   .replace('...', '... `[STATUS: SECURE]`\n');
    }

    if (aiMessageIndex < messages.value.length) {
      messages.value[aiMessageIndex].content += formattedData
    }
  }

  const onError = (error) => {
    console.error('Agent SSE Error:', error)
    connectionStatus.value = 'error'
    if (aiMessageIndex < messages.value.length && messages.value[aiMessageIndex].content === '') {
       messages.value[aiMessageIndex].content = '`[FATAL] CONNECTION_REFUSED: 无法建立与底层 MCP 中枢的安全连接。`'
    }
  }
  
  eventSource = await chatWithThreatHunter(message, chatId.value, onMessage, onError)
}

const goBack = () => {
  router.push('/')
}

onMounted(() => {
  chatId.value = generateChatId()
  // 匹配深色风格的全新启动提示语
  const bootMessage = `**[SYSTEM BOOT] ZeroTrust Sentinel 核心组建加载完毕。**\n\n> ThreatHunter (威胁狩猎节点) 已上线。\n> 底层 MCP 协议栈已就绪，沙箱环境 [隔离等级: S 级] 已激活。\n\n**请下发操作指令：**\n1. 提交需研判的异常流量特征\n2. 查询本地恶意样本特征库\n3. 验证特定的 CVE 漏洞利用逻辑`
  addMessage(bootMessage, false)
})

onBeforeUnmount(() => {
  if (eventSource && typeof eventSource.close === 'function') {
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

/* ================= 布局核心设置 ================= */
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

/* 头部样式 */
.header {
  flex-shrink: 0; /* 保证头部不被挤压 */
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
  letter-spacing: 1px;
}

.accent-text {
  color: var(--text-dim);
  font-weight: normal;
  font-size: 14px;
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

/* ================= 聊天内容区 ================= */
.content-wrapper, .chat-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;      
}

/* ================= 强行修正底层 ChatRoom 样式，使其贴合新主题 ================= */
:deep(.chat-container) {
  border: none !important; /* 去掉边框 */
  background-color: transparent !important;
  border-radius: 0 !important;
  box-shadow: none !important;
}

:deep(.chat-input-container) {
  flex-shrink: 0 !important;
}

:deep(.chat-messages) {
  flex: 1 !important;
  height: 0 !important;
  overflow-y: auto !important;
}

/* 修改底层输入框及按钮的主题色，从刺眼绿变为极光蓝 */
:deep(.prompt-prefix) { color: var(--accent-blue) !important; }
:deep(.input-box) { color: var(--text-main) !important; }
:deep(.send-button) { color: var(--accent-blue) !important; }
:deep(.send-button:hover:not(:disabled)) { 
  background-color: rgba(88, 166, 255, 0.1) !important; 
  text-shadow: 0 0 5px var(--accent-blue) !important; 
}
:deep(.ai-message .message-bubble) { 
  background-color: rgba(88, 166, 255, 0.05) !important; 
  border: 1px solid var(--border-color) !important; 
  color: var(--text-main) !important; 
}
:deep(.user-message .message-bubble) { 
  border-color: var(--accent-blue) !important; 
  color: var(--accent-blue) !important; 
}

/* 响应式 */
@media (max-width: 768px) {
  .header { padding: 12px 16px; }
  .title { font-size: 14px; }
  .chat-id { display: none; }
}
</style>