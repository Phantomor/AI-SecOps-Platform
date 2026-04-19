package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sentinel-agent-go/internal/config"
)

// DashScopeResponse 阿里云向量接口返回结构
type DashScopeResponse struct {
	Output struct {
		Embeddings []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"embeddings"`
	} `json:"output"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// GenerateEmbedding 调用通义千问 (text-embedding-v2) 将文本转为 1536 维向量
func GenerateEmbedding(text string) ([]float32, error) {
	// 核心优雅改造：直接从你的 application.yaml 中读取 API Key
	apiKey := config.Global.AI.OpenAI.APIKey

	if apiKey == "" || apiKey == "sk-xxx" {
		return nil, fmt.Errorf("🚨 请在 application.yaml 中填入真实的通义千问 API Key")
	}

	url := "https://dashscope.aliyuncs.com/api/v1/services/embeddings/text-embedding/text-embedding"

	reqBody := map[string]interface{}{
		"model": "text-embedding-v2",
		"input": map[string]interface{}{
			"texts": []string{text},
		},
		"parameters": map[string]interface{}{
			"text_type": "query",
		},
	}
	bodyData, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	var result DashScopeResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析千问响应失败: %v", err)
	}

	if result.Code != "" {
		return nil, fmt.Errorf("通义千问 API 报错: %s - %s", result.Code, result.Message)
	}

	if len(result.Output.Embeddings) == 0 {
		return nil, fmt.Errorf("未获取到 Embedding 结果")
	}

	return result.Output.Embeddings[0].Embedding, nil
}
