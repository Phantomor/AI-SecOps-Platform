# ZeroTrust Sentinel (AI-SecOps-Platform)

**ZeroTrust Sentinel** 是一款面向企业级安全运营（SecOps）的下一代 AI 智能体平台。它不仅是一个对话机器人，更是一个具备**环境感知、自主调度、合规研判**能力的“数字安全专家”。

本项目采用 **Go 语言构建高性能核心引擎**，配合 **Vue 3** 极客终端前端，深度集成了 **RAG（检索增强生成）**、**ReAct 智能体范式** 与 **MCP（模型上下文协议）**，实现了从底层威胁数据穿透到高层合规审计的全链路闭环。

------

##  核心特性

- **智能威胁狩猎 (ThreatHunter Agent)**: 基于 Go 语言与 ReAct 引擎，自主调度 MCP 工具提取本地 SQLite 流量告警库，实现对 Android 恶意软件及概念漂移流量的精准研判。
- **语义级 RAG 知识库 (Compliance Copilot)**: 原生集成 **Elasticsearch 8.x 向量检索 (Dense Vector)**，利用通义千问 `text-embedding-v1` 实现企业安全规范文档（Markdown）的毫秒级召回与防幻觉问答。
- **MCP 协议驱动架构**: 采用 **Model Context Protocol** 标准，Go 后端作为跨系统网关，支持动态挂载外部安全武器库（本地数据库查询、网络搜索、代码沙箱），实现 AI 推理与底层系统的绝对解耦。
- **自动化 Skill SOP (技能流)**: 将高频安全研判逻辑（如高危 IP 分析）封装为代码级 SOP 工作流。大模型一次调用，后端 Go 协程高并发执行“外部情报查询 -> ES 内部规范检索 -> 终端日志融合”，大幅提升研判效率并降低 Token 消耗。
- **极致的 SSE 流式体验**: 摒弃传统的 HTTP 阻塞等待，前后端采用 Server-Sent Events (SSE) 建立长连接，实时渲染 AI 的 `[AUDIT LOG] 思考过程` 与工具调度状态，提供全息沉浸式交互。

------

## 系统架构

项目采用“高并发控制面与 AI 调度引擎”结合的设计：

- **Control Plane (Go / Gin)**: 负责处理海量 WebSocket / SSE 并发连接，利用 Redis 维护滑动窗口式的 Agent 长期记忆与分布式会话。
- **AI Agent Engine (Go / Eino)**: 深度二次封装大模型框架，实现 `RunReAct` 统一调度中心。通过注入不同的 System Prompt 瞬间衍生出不同职能的数字安全专家。
- **Data Layer**:
  - **Elasticsearch 8.x**: 负责非结构化安全文档的 `kNN` 向量存储与召回。
  - **SQLite/MySQL**: 存储结构化网络告警日志与终端指纹。
  - **Redis**: 维持高并发场景下的 Agent 上下文与分布式锁。

------

## 技术栈

| **模块**            | **技术方案**                                                 |
| ------------------- | ------------------------------------------------------------ |
| **核心后端 (Go)**   | Golang, Gin, Eino (ByteDance LLM Framework), GORM            |
| **极客前端 (Vue)**  | Vue 3, Vite, Tailwind CSS, SSE Client                        |
| **AI / RAG 引擎**   | 通义千问 (Qwen-plus API), ES 8.x (Dense Vector), text-embedding-v2 |
| **基础设施 & 存储** | Docker Compose, Elasticsearch, Redis, SQLite                 |

------

## 快速搭建

### 1. 环境准备

- Go 1.21+ / Node.js 18+
- Docker & Docker Compose
- 申请 [阿里云百炼](https://bailian.console.aliyun.com/) API Key

### 2. 部署基础组件

在项目根目录启动 Elasticsearch 与 Redis：

```Bash
docker-compose up -d
```

### 3. 配置环境变量

修改 `configs/application.yaml` 配置相关中间件地址，并设置千问 API 密钥.

### 4. 注入向量知识库 (ETL Pipeline)

将企业的安全规范文档放入 `assets/knowledge/` 目录，并执行初始化脚本：

```Bash
go run cmd/script/init_es_data.go
```

*(脚本将自动调用 Embedding API 并写入 ES 8.x 向量库)*

### 5. 启动引擎

```Bash
# 启动 Go 后端守护进程
go run cmd/server/main.go

# 新开终端启动 Vue 3 前端
cd sentinel-terminal
npm install && npm run dev
```

------

## 核心实战场景

### 场景一：DevSecOps 合规审查 (RAG 增强)

- **输入**：“公司针对高并发接口重放攻击有什么强制规范？”
- **流转**：Agent 拦截提问 $\rightarrow$ 动态挂载 `ext_knowledge_search` 工具 $\rightarrow$ 提取向量 $\rightarrow$ Elasticsearch kNN 检索 $\rightarrow$ 结合内部《Go 语言安全编码红线》输出防御方案。

### 场景二：自动化安全研判 (Skill SOP)

- **输入**：“研判外部 IP 185.241.165.12，并告知处置流程。”
- **流转**：Agent 识别高频任务 $\rightarrow$ 挂载原生技能 `skill_ip_triage` $\rightarrow$ Go 后端执行原子调度（微步情报 + ES内部规范 + WAF关联日志） $\rightarrow$ 聚合输出完整安全报告。

------

## 许可证

本项目采用 [Apache License 2.0](https://www.google.com/search?q=LICENSE) 许可证。