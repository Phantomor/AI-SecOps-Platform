package dto

// ChatRequest 基础对话请求参数
type ChatRequest struct {
	Message string `json:"message" binding:"required" example:"什么是 GMP 模型？"`
}

// StreamChatRequest 流式对话请求参数
type StreamChatRequest struct {
	SessionID string `json:"session_id" binding:"required" example:"user_123_session"`
	Message   string `json:"message" binding:"required" example:"请解释一下 Go 的垃圾回收机制。"`
}

// ReportRequest 面试报告生成请求参数
type ReportRequest struct {
	SessionID string `json:"session_id" binding:"required" example:"user_123_session"`
	Resume    string `json:"resume" binding:"required" example:"熟练掌握Go语言，具有高并发经验..."`
	JD        string `json:"jd" binding:"required" example:"需要3年Go开发经验，熟悉GMP模型..."`
}

// InterviewReport 面试报告返回结果结构
type InterviewReport struct {
	Score          int      `json:"score" example:"85"`
	Strengths      []string `json:"strengths" example:"熟悉 Go 并发,回答逻辑清晰"`
	Weaknesses     []string `json:"weaknesses" example:"对底层调度理解不深"`
	HireConclusion string   `json:"hire_conclusion" example:"Hire"`
	Comments       string   `json:"comments" example:"该候选人基础扎实，符合岗位要求..."`
}

// ThreatHunterRequest 威胁狩猎专家对话请求参数
type ThreatHunterRequest struct {
	SessionID string `json:"session_id" binding:"required" example:"threat_hunter_123_session"`
	Message   string `json:"message" binding:"required" example:"帮我查一下这名候选人的背景信息"`
}
