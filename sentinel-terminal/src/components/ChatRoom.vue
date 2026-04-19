<template>
  <div class="chat-container">
    <div class="chat-messages" ref="messagesContainer" @click="handleChatClick">
      <div v-for="(msg, index) in messages" :key="index" class="message-wrapper">
        <div v-if="!msg.isUser" class="message ai-message" :class="[msg.type]">
          <div class="avatar ai-avatar">
            <div class="avatar-placeholder ai-icon">SYS</div>
          </div>
          <div class="message-bubble">
            <div 
              class="message-content markdown-body selectable-text" 
              v-html="renderMarkdown(msg.content, index === messages.length - 1 && connectionStatus === 'connecting')"
            ></div>
            <div class="message-time">{{ formatTime(msg.time) }}</div>
          </div>
        </div>
        
        <div v-else class="message user-message" :class="[msg.type]">
          <div class="message-bubble">
            <div class="message-content selectable-text">>_ {{ msg.content }}</div>
            <div class="message-time">{{ formatTime(msg.time) }}</div>
          </div>
          <div class="avatar user-avatar">
            <div class="avatar-placeholder admin-icon">ADM</div>
          </div>
        </div>
      </div>
    </div>

    <div class="chat-input-container">
      <div class="stop-btn-wrapper" v-if="connectionStatus === 'connecting'">
        <button class="stop-btn" @click="$emit('stop-generating')">
          <span class="stop-icon">✖</span> ABORT_PROCESS
        </button>
      </div>

      <div class="chat-input">
        <span class="prompt-prefix">root@ZeroTrust:~#</span>
        <textarea 
          v-model="inputMessage" 
          @keydown.enter.prevent="sendMessage"
          placeholder="请输入探测指令或查询条件..." 
          class="input-box"
          :disabled="connectionStatus === 'connecting'"
        ></textarea>
        <button 
          @click="sendMessage" 
          class="send-button"
          :disabled="connectionStatus === 'connecting' || !inputMessage.trim()"
        >EXECUTE</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, nextTick, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { marked } from 'marked'
import hljs from 'highlight.js'
// 移除 github.css，改用暗黑主题高亮，如果你安装了 hljs，建议用 atom-one-dark 或直接依赖我们的自定义 CSS
// import 'highlight.js/styles/atom-one-dark.css' 

const renderer = new marked.Renderer()

renderer.code = function (code, lang) {
  const language = hljs.getLanguage(lang) ? lang : 'plaintext'
  const highlighted = hljs.highlight(code, { language }).value
  const encodedCode = encodeURIComponent(code)
  
  return `
    <div class="code-block-wrapper">
      <div class="code-header">
        <span class="code-lang">[${lang || 'CODE_BLOCK'}]</span>
        <button class="copy-btn" data-code="${encodedCode}">COPY_DATA</button>
      </div>
      <pre><code class="hljs ${language}">${highlighted}</code></pre>
    </div>
  `
}

marked.setOptions({
  renderer: renderer,
  breaks: true, 
})

const props = defineProps({
  messages: { type: Array, default: () => [] },
  connectionStatus: { type: String, default: 'disconnected' },
  aiType: { type: String, default: 'default' }
})

const emit = defineEmits(['send-message', 'stop-generating'])
const inputMessage = ref('')
const messagesContainer = ref(null)

const handleChatClick = (e) => {
  if (e.target.classList.contains('copy-btn')) {
    const rawCode = decodeURIComponent(e.target.getAttribute('data-code'))
    
    navigator.clipboard.writeText(rawCode).then(() => {
      const btn = e.target
      const originalText = btn.innerText
      btn.innerText = 'COPIED_OK ✓'
      btn.classList.add('copied')
      ElMessage({ message: '数据已安全复制到剪贴板', type: 'success', customClass: 'cyber-toast' })
      
      setTimeout(() => {
        btn.innerText = originalText
        btn.classList.remove('copied')
      }, 2000)
    }).catch(err => {
      ElMessage.error('权限不足，复制失败')
      console.error('复制失败:', err)
    })
  }
}

const renderMarkdown = (text, isTyping) => {
  if (!text) return isTyping ? '<span class="typing-indicator">_</span>' : ''
  let html = marked.parse(text)
  if (isTyping) {
    if (html.endsWith('</p>\n')) {
      html = html.slice(0, -5) + '<span class="typing-indicator">_</span></p>\n'
    } else {
      html += '<span class="typing-indicator">_</span>'
    }
  }
  return html
}

const sendMessage = () => {
  if (!inputMessage.value.trim()) return
  emit('send-message', inputMessage.value)
  inputMessage.value = ''
}

const formatTime = (timestamp) => {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

const scrollToBottom = async () => {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

watch(() => props.messages.length, scrollToBottom)
watch(() => props.messages.map(m => m.content).join(''), scrollToBottom)

onMounted(scrollToBottom)
</script>

<style scoped>
/* ================== 全局与容器 ================== */
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background-color: #050505; /* 纯黑背景 */
  border: 1px solid #00ff00; /* 绿色发光边框 */
  box-shadow: 0 0 15px rgba(0, 255, 0, 0.1) inset;
  border-radius: 4px;
  overflow: hidden;
  position: relative;
  font-family: 'Fira Code', 'JetBrains Mono', Consolas, monospace;
}

.chat-messages {
  flex: 1;
  height: 0;
  overflow-y: auto;
  padding: 20px 20px 40px 20px;
  display: flex;
  flex-direction: column;
  scrollbar-width: thin;
  scrollbar-color: #00ff00 #050505;
}
.chat-messages::-webkit-scrollbar { width: 6px; }
.chat-messages::-webkit-scrollbar-thumb { background: #004400; border-radius: 3px; }

/* ================== 消息气泡 ================== */
/* 1. 确保包装器是 Flex 容器，以便控制子元素的对齐 */
.message-wrapper { 
  display: flex;
  width: 100%; 
  margin-bottom: 20px; 
  flex-shrink: 0;
}

/* 2. 通用消息样式，限制最大宽度 */
.message { 
  display: flex; 
  align-items: flex-start; 
  max-width: 85%; /* 稍微缩小一点，给对侧留出空间 */
  margin-bottom: 8px; 
}

/* 3. AI (Agent) 消息：固定在左侧 */
.ai-message { 
  margin-right: auto; /* 确保它贴向左边 */
  justify-content: flex-start;
}

/* 4. 用户 (User) 消息：强制推向右侧 */
.user-message { 
  margin-left: auto; /* 关键：将整个消息块推向容器右侧 */
  flex-direction: row; /* [气泡] [头像] 的顺序 */
  justify-content: flex-end;
}

.avatar { width: 36px; height: 36px; flex-shrink: 0; }
.user-avatar { margin-left: 12px; }
.ai-avatar { margin-right: 12px; }

.avatar-placeholder { 
  width: 100%; height: 100%; 
  display: flex; align-items: center; justify-content: center; 
  font-weight: bold; font-size: 12px; border-radius: 4px;
  border: 1px solid currentColor;
}
.admin-icon { color: #00f0ff; border-color: #00f0ff; background: rgba(0, 240, 255, 0.1); }
.ai-icon { color: #00ff00; border-color: #00ff00; background: rgba(0, 255, 0, 0.1); }

.message-bubble {
  padding: 12px 16px; 
  border-radius: 2px;
  position: relative;
  word-wrap: break-word;
  word-break: break-word; /* ：允许在单词内换行 */
  min-width: 150px; 
  max-width: 100%;        /* ：限制最大宽度不超过父级 */
  overflow-x: auto;       /* ：超长内容在气泡内部出现滚动条 */
  line-height: 1.6;
}
/* 用户指令泡泡 */
.user-message .message-bubble { 
  background-color: rgba(0, 240, 255, 0.05); 
  color: #00f0ff; 
  border: 1px solid #00f0ff; 
  box-shadow: 0 0 10px rgba(0, 240, 255, 0.1);
}

/* AI 响应泡泡 */
.ai-message .message-bubble { 
  background-color: rgba(0, 255, 0, 0.02); 
  border: 1px solid #004400; 
  color: #00ff00; 
  box-shadow: 0 0 10px rgba(0, 255, 0, 0.05);
}

.message-time { font-size: 11px; opacity: 0.5; margin-top: 8px; text-align: right; }

/* ================== 底部输入区 ================== */
.chat-input-container { 
  position: relative; 
  flex-shrink: 0;
  background-color: #0a0a0a; 
  border-top: 1px solid #00ff00; 
  padding: 16px 20px;
}

.chat-input { 
  display: flex; 
  align-items: center;
  background-color: #000;
  border: 1px solid #004400;
  border-radius: 4px;
  padding: 4px 12px;
}

.prompt-prefix {
  color: #00ff00;
  font-weight: bold;
  margin-right: 8px;
  user-select: none;
}

.input-box { 
  flex-grow: 1; 
  background: transparent;
  color: #00ff00;
  border: none;
  padding: 8px 0; 
  font-size: 14px; 
  font-family: inherit;
  resize: none; 
  height: 40px; 
  outline: none; 
}
.input-box::placeholder { color: rgba(0, 255, 0, 0.3); }

.send-button { 
  margin-left: 12px; 
  background-color: transparent; 
  color: #00ff00; 
  border: 1px solid #00ff00; 
  border-radius: 4px; 
  padding: 6px 16px; 
  font-family: inherit;
  font-weight: bold;
  cursor: pointer; 
  transition: all 0.2s; 
}
.send-button:hover:not(:disabled) { 
  background-color: #00ff00; 
  color: #000; 
  box-shadow: 0 0 10px #00ff00;
}
.input-box:disabled, .send-button:disabled { opacity: 0.4; cursor: not-allowed; border-color: #004400; color: #004400;}

/* ================== ABORT 按钮 ================== */
.stop-btn-wrapper {
  position: absolute;
  top: -45px;
  left: 0;
  width: 100%;
  display: flex;
  justify-content: center;
  pointer-events: none; 
}

.stop-btn {
  pointer-events: auto; 
  background-color: #1a0000;
  border: 1px solid #ff0000;
  color: #ff0000;
  padding: 6px 20px;
  font-family: inherit;
  font-weight: bold;
  font-size: 12px;
  cursor: pointer;
  box-shadow: 0 0 10px rgba(255, 0, 0, 0.3);
  display: flex;
  align-items: center;
  gap: 8px;
  transition: all 0.2s;
  letter-spacing: 1px;
}
.stop-btn:hover { background-color: #ff0000; color: #000; box-shadow: 0 0 20px #ff0000; }

.typing-indicator { display: inline-block; animation: blink 1s step-end infinite; margin-left: 2px; font-weight: bold; color: #00ff00;}
@keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }

/* ================== Markdown 与代码块美化 (极客版) ================== */
.selectable-text { user-select: text !important; cursor: text; }

:deep(.markdown-body) { font-size: 14px; line-height: 1.6; color: #00ff00; }
:deep(.markdown-body p) { margin: 0 0 10px 0; }

/* 赛博朋克表格 */
:deep(.markdown-body table) {
  display: block;         /* ：将表格变为块级容器 */
  overflow-x: auto;       /* ：允许表格横向滑动 */
  border-collapse: collapse;
  width: 100%;
  max-width: 100%;        /* ：最大宽度限制 */
  margin: 15px 0;
  background-color: rgba(0, 20, 0, 0.3);
  /* 自定义表格的极客滚动条 */
  scrollbar-width: thin;
  scrollbar-color: #00ff41 transparent;
}

/* （可选）给表格滚动条加点科技感 */
:deep(.markdown-body table::-webkit-scrollbar) {
  height: 6px;
}
:deep(.markdown-body table::-webkit-scrollbar-thumb) {
  background: #004411;
  border-radius: 3px;
}

:deep(.markdown-body th), :deep(.markdown-body td) {
  white-space: nowrap;
  border: 1px solid #004400;
  padding: 8px 12px;
  text-align: left;
}
:deep(.markdown-body th) { background-color: #002200; color: #00ff00; font-weight: bold; text-transform: uppercase; }

/* 赛博朋克代码块 */
:deep(.code-block-wrapper) {
  background-color: #000;
  border: 1px solid #00f0ff;
  border-radius: 4px;
  margin: 15px 0;
  box-shadow: 0 0 8px rgba(0, 240, 255, 0.1);
}
:deep(.code-header) {
  display: flex; justify-content: space-between; align-items: center;
  background-color: rgba(0, 240, 255, 0.1);
  padding: 6px 12px; font-size: 12px; color: #00f0ff;
  border-bottom: 1px solid #00f0ff;
}
:deep(.copy-btn) {
  background: transparent; border: 1px solid #00f0ff; color: #00f0ff;
  padding: 2px 8px; font-family: inherit; font-size: 11px; cursor: pointer; transition: 0.2s;
}
:deep(.copy-btn:hover) { background: #00f0ff; color: #000; box-shadow: 0 0 5px #00f0ff; }
:deep(.copy-btn.copied) { background: #00ff00; color: #000; border-color: #00ff00; }

:deep(.code-block-wrapper pre) { margin: 0; padding: 12px; overflow-x: auto; }
:deep(.markdown-body code) { font-family: inherit; color: #00f0ff; }
:deep(.markdown-body p > code), :deep(.markdown-body li > code) {
  background-color: rgba(0, 240, 255, 0.1);
  padding: 2px 4px; border: 1px solid #004466; border-radius: 2px; color: #00f0ff;
}
:deep(.markdown-body strong) { font-weight: bold; color: #00f0ff; text-shadow: 0 0 2px #00f0ff; }
</style>