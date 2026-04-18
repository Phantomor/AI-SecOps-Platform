import axios from 'axios'

const API_BASE_URL = import.meta.env.PROD 
 ? '/api' 
 : 'http://127.0.0.1:8080/api'

// === 基础 Axios 配置保持不变 ===
const request = axios.create({
  baseURL: API_BASE_URL,
  timeout: 60000,
  withCredentials: true 
})

request.interceptors.request.use(config => config, error => Promise.reject(error))
request.interceptors.response.use(
  response => {
    if (response.data.code === 200 || response.data.code === 0) {
      return response.data.data
    } else {
      throw new Error(response.data.message || '请求失败')
    }
  },
  error => Promise.reject(error)
)

// === 🚀 核心修复：非阻塞式的 SSE 流式接收函数 ===
export const connectSSE = (url, params, onMessage, onError) => {
  const fullUrl = `${API_BASE_URL}${url}`
  const abortController = new AbortController()

  // 让 fetch 在后台默默执行，不要阻塞主线程
  fetch(fullUrl, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(params),
    signal: abortController.signal
  }).then(async (response) => {
    if (!response.ok) throw new Error(`HTTP 异常: ${response.status}`);
    
    const reader = response.body.getReader();
    const decoder = new TextDecoder('utf-8');
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      // 🌟 核心修复：如果底层 TCP 连接结束了，无论有没有收到 [DONE]，都强制关闭光标
      if (done) {
        if (onMessage) onMessage('[DONE]');
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n\n');
      buffer = lines.pop(); // 保留不完整的块

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const dataStr = line.substring(6);
          
          if (dataStr === '[DONE]') {
            if (onMessage) onMessage('[DONE]');
            return; // 结束执行
          }
          
          try {
            // 解析 Go 后端发来的 {"content": "..."}
            const parsed = JSON.parse(dataStr);
            if (onMessage) onMessage(parsed.content || '');
          } catch (e) {
            if (onMessage) onMessage(dataStr);
          }
        }
      }
    }
  }).catch(error => {
    if (error.name !== 'AbortError' && onError) onError(error);
  });

  // 立即返回控制器，Vue 就能立刻挂载它！
  return {
    close: () => abortController.abort()
  }
}

// === 🚀 确保这几个回调参数 (onMessage, onError) 完美透传 ===
// DevSecOps 合规顾问聊天
export const chatWithComplianceCopilot = (message, sessionId, onMessage, onError) => {
  // 注意：如果你的 connectSSE 底层自动拼接了 '/api'，这里就写 '/secops/compliance'
  // 如果没有拼接，请写完整路径 '/api/secops/compliance'
  return connectSSE('/secops/compliance', { message, session_id: sessionId }, onMessage, onError)
}

// ThreatHunter 威胁狩猎专家聊天
export const chatWithThreatHunter = (message, sessionId, onMessage, onError) => {
  // 对应后端新路由: /secops/threat_hunter
  return connectSSE('/secops/threat_hunter', { message, session_id: sessionId }, onMessage, onError)
}

// 用户模块 (保持不变)
export const userRegister = (u, p, c) => request.post('/user/register', { userAccount: u, userPassword: p, checkPassword: c })
export const userLogin = (u, p) => request.post('/user/login', { userAccount: u, userPassword: p })
export const getLoginUser = () => request.get('/user/get/login')
export const userLogout = () => request.post('/user/logout')

export default {
  chatWithComplianceCopilot, chatWithThreatHunter, userRegister, userLogin, getLoginUser, userLogout
}