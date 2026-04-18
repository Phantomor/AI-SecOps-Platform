// internal/agent/registry.go
package agent

import (
	"fmt"
	"sync"

	"github.com/cloudwego/eino/schema"
)

// Tool 定义了 AI 插件必须实现的标准化接口
type Tool interface {
	// Info 返回给大模型的工具说明书 (Schema)
	Info() *schema.ToolInfo
	// Execute 执行工具的具体逻辑，入参是 JSON 格式的字符串
	Execute(args string) (string, error)
}

// Registry 插件注册表 (并发安全)
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// GlobalRegistry 全局唯一的工具注册表
var GlobalRegistry = &Registry{
	tools: make(map[string]Tool),
}

// Register 注册一个新工具
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Info().Name] = tool
}

// GetTool 获取指定工具
func (r *Registry) GetTool(name string) (Tool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if tool, ok := r.tools[name]; ok {
		return tool, nil
	}
	return nil, fmt.Errorf("tool [%s] not found", name)
}

// GetAllToolInfos 获取所有已注册工具的 Schema，用于发给大模型
func (r *Registry) GetAllToolInfos() []*schema.ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]*schema.ToolInfo, 0, len(r.tools))
	for _, t := range r.tools {
		infos = append(infos, t.Info())
	}
	return infos
}
