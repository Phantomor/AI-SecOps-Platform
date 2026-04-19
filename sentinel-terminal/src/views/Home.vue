<template>
  <div class="terminal-container">
    <div class="bg-overlay"></div>
    
    <div class="hero-section">
      <div class="header-content">
        <div class="boot-sequence">
          <p>[INIT] ZeroTrust Sentinel Kernel 2.0.4...</p>
          <p>[OK] Secure Layer v4 Active.</p>
        </div>
        
        <h1 class="terminal-title">
          Sentinel <span class="accent-text">SecOps</span>
        </h1>
        <p class="subtitle">guest@sentinel:~$ <span class="typing">./list_modules</span><span class="cursor">_</span></p>
      </div>

      <div class="scroll-indicator">
        <span class="scroll-text">SYSTEM_READY // SCROLL_DOWN</span>
        <div class="scroll-arrow">↓</div>
      </div>
    </div>

    <div class="apps-container">
      <div class="terminal-card" @click="navigateTo('/threat-hunter')">
        <div class="card-tag">INTEL_01</div>
        <div class="app-info">
          <div class="app-title">Threat Hunter</div>
          <p class="app-desc">自动化威胁追捕与流量研判中心。基于 MCP 协议接入本地沙箱。</p>
        </div>
        <div class="card-footer">CONNECT_NODE >></div>
      </div>
      
      <div class="terminal-card" @click="navigateTo('/compliance-copilot')">
        <div class="card-tag">AUDIT_02</div>
        <div class="app-info">
          <div class="app-title">Compliance Copilot</div>
          <p class="app-desc">DevSecOps 合规顾问。提供安全编码审查、漏洞修复建议与合规性核查。</p>
        </div>
        <div class="card-footer">CONNECT_NODE >></div>
      </div>
    </div>
    
    <AppFooter class="terminal-footer" />
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router'
import { useHead } from '@vueuse/head'
import AppFooter from '../components/AppFooter.vue'

useHead({
  title: 'ZeroTrust Sentinel | AI SecOps',
  meta: [
    {
      name: 'description',
      content: 'ZeroTrust Sentinel - 零信任哨兵 AI SecOps 平台。'
    }
  ]
})

const router = useRouter()

const navigateTo = (path) => {
  router.push(path)
}
</script>

<style scoped>
:root {
  --bg-deep: #0a0c10;      
  --bg-card: #161b22;      
  --accent-blue: #58a6ff;  
  --text-main: #c9d1d9;    
  --text-dim: #8b949e;     
  --border-color: #30363d; 
}

.home-container {
  display: flex;
  flex-direction: column;
  min-height: 100vh; /* 确保背景至少铺满一屏 */
  height: auto;      /* 允许高度根据内容自动增加 */
  overflow-y: visible; /* 确保首页可以正常滚动 */
  background-color: var(--bg-deep);
}

.terminal-container {
  /* 移除之前的 padding，让 hero-section 能精准计算 100vh */
  width: 100%;
  background-color: var(--bg-deep);
  color: var(--text-main);
  font-family: 'Fira Code', 'Inter', monospace;
  position: relative;
}

/* 背景层：改成 fixed，永远填满浏览器可视区域 */
.bg-overlay {
  position: fixed;
  top: 0; left: 0; width: 100vw; height: 100vh;
  background: radial-gradient(circle at 50% 0%, #1f2937 0%, transparent 70%);
  pointer-events: none;
  z-index: 0;
}

/* 首屏霸屏容器：高度 = 屏幕高度 - 顶部导航栏(假设60px) */
.hero-section {
  min-height: calc(100vh - 60px);
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center; /* 居中对齐 */
  position: relative;
  z-index: 1;
  padding: 20px;
}

.header-content {
  max-width: 900px;
  width: 100%;
  /* 稍微往上提一点，让视觉重心更舒适 */
  transform: translateY(-10%); 
}

.boot-sequence {
  font-size: 0.9rem;
  color: var(--text-dim);
  margin-bottom: 25px;
}

.terminal-title {
  font-size: 3.5rem;
  font-weight: bold;
  color: #ffffff;
  margin: 10px 0;
  letter-spacing: -1px;
}

.accent-text {
  color: var(--accent-blue);
  text-shadow: 0 0 15px rgba(88, 166, 255, 0.3);
}

.subtitle {
  font-size: 1.2rem;
  color: var(--text-dim);
  margin-top: 15px;
}

.typing {
  color: var(--text-main);
}

.cursor {
  display: inline-block;
  width: 10px;
  height: 1.2rem;
  background-color: var(--accent-blue);
  vertical-align: bottom;
  margin-left: 2px;
  animation: blink 1s step-end infinite;
}

/* 动态向下提示 */
.scroll-indicator {
  position: absolute;
  bottom: 40px; /* 距离屏幕底部 40px */
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  opacity: 0.6;
  transition: opacity 0.3s;
}

.scroll-indicator:hover {
  opacity: 1;
}

.scroll-text {
  font-size: 0.75rem;
  letter-spacing: 2px;
  color: var(--accent-blue);
  margin-bottom: 8px;
}

.scroll-arrow {
  color: var(--accent-blue);
  font-size: 1.2rem;
  animation: bounce 2s infinite;
}

/* 卡片容器：被推到第二屏 */
.apps-container {
  display: flex;
  gap: 30px;
  max-width: 1000px;
  margin: 40px auto 100px; /* 底部留白 */
  padding: 0 20px;
  position: relative;
  z-index: 1;
}

.terminal-card {
  flex: 1;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 30px;
  cursor: pointer;
  transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
  display: flex;
  flex-direction: column;
}

.terminal-card:hover {
  border-color: var(--accent-blue);
  background: #1c2128;
  transform: translateY(-8px);
  box-shadow: 0 12px 40px rgba(0,0,0,0.5);
}

.card-tag {
  font-size: 0.75rem;
  color: var(--accent-blue);
  letter-spacing: 1.5px;
  margin-bottom: 20px;
  font-weight: bold;
}

.app-info {
  flex-grow: 1;
}

.app-title {
  font-size: 1.6rem;
  font-weight: 600;
  margin-bottom: 15px;
  color: #f0f6fc;
}

.app-desc {
  color: var(--text-dim);
  font-size: 0.95rem;
  line-height: 1.7;
}

.card-footer {
  margin-top: 40px;
  font-size: 0.85rem;
  font-weight: bold;
  color: var(--accent-blue);
  opacity: 0.7;
  transition: opacity 0.2s;
}

.terminal-card:hover .card-footer {
  opacity: 1;
}

.terminal-footer {
  position: relative;
  z-index: 1;
}

/* 动画 */
@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

@keyframes bounce {
  0%, 20%, 50%, 80%, 100% { transform: translateY(0); }
  40% { transform: translateY(8px); }
  60% { transform: translateY(4px); }
}

/* 响应式 */
@media (max-width: 768px) {
  .terminal-title {
    font-size: 2.5rem;
  }
  .apps-container { 
    flex-direction: column; 
  }
  .header-content {
    transform: translateY(-15%);
  }
}
</style>