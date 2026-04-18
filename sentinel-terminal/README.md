# ZeroTrust Sentinel | AI SecOps Terminal

**ZeroTrust Sentinel** 是一个基于极客终端（Cyber Terminal）美学设计的 AI 赋能安全运营平台。它通过高度集成的智能体，为安全专家提供威胁狩猎与合规自动化能力。

## 🛠 核心作战单元 (Core Agents)

- **⚡ ThreatHunter (威胁猎人)** 基于大语言模型的实时威胁情报分析与狩猎工具。支持模拟渗透路径分析、恶意代码行为解释及响应建议。
- **🛡️ Compliance Copilot (合规副驾)** 自动化合规审计助手。针对多行业标准（如等保2.0、GDPR）提供自动化的策略核查与补救建议。

## 🚀 技术架构 (Tech Stack)

- **Runtime**: Vue 3 (Composition API)
- **Engine**: Vite - 极速构建与热更新
- **Streaming**: SSE (Server-Sent Events) - 实现流式指令输出，模拟真实终端体感
- **Networking**: Axios + 拦截器（模拟零信任鉴权逻辑）
- **UI/UX**: 纯手工打造的 CSS 极客风格，支持 `fixed` 视口锁定与内部滚动容器（Gemini 交互模式）

## 🖥️ 快速部署

### 环境要求

- Node.js >= 18.0.0
- NPM >= 8.0.0

### 初始化指令

```Bash
# 克隆仓库
git clone <repository-url>

# 进入终端目录
cd sentinel-terminal

# 安装加密依赖
npm install
```

### 启动终端

```Bash
npm run dev
```

### 编译生产环境镜像

```Bash
npm run build
```

## 📡 链路配置

系统默认通过 `Vite Proxy` 或环境变量连接后端安全网关：

- **Gateway Address**: `http://localhost:5173/api` (开发环境)
- **Stream Interface**: `/api/chat/stream`

## ⚖️ 开源协议

本项目采用 MIT 协议。仅限用于合法的网络安全研究与防御性运营。