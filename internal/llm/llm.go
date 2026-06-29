// Package llm provides an OpenAI-compatible LLM client for chat completion.
// Supports any OpenAI-compatible API (Grok, DeepSeek, Ollama, OpenAI, etc.).
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Message represents a chat message in the request.
type Message struct {
	Role    string `json:"role"`    // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatRequest is the standard OpenAI chat completion request body.
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// ChatResponse is the standard OpenAI chat completion response body.
type ChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int     `json:"index"`
		Message Message `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// Client is an LLM API client.
type Client struct {
	Endpoint   string
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

// NewClient creates a new LLM client.
// Endpoint examples:
//   - https://api.x.ai/v1           (Grok)
//   - https://api.deepseek.com/v1   (DeepSeek)
//   - http://localhost:11434/v1      (Ollama local)
//   - https://api.openai.com/v1     (OpenAI)
func NewClient(endpoint, apiKey, model string) *Client {
	return &Client{
		Endpoint:   endpoint,
		APIKey:     apiKey,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 120 * time.Second},
	}
}

// Chat sends a chat completion request and returns the response.
func (c *Client) Chat(req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.3
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.Endpoint+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	}

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	return &chatResp, nil
}

// SimpleChat is a convenience method for a single-turn chat.
func (c *Client) SimpleChat(system, user string) (string, error) {
	req := ChatRequest{
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}
	resp, err := c.Chat(req)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return resp.Choices[0].Message.Content, nil
}

// ProviderDefaults returns default endpoint and model for known providers.
func ProviderDefaults(provider string) (endpoint, model string) {
	switch provider {
	case "grok", "xai":
		return "https://api.x.ai/v1", "grok-4-20-0309-reasoning"
	case "deepseek":
		return "https://api.deepseek.com/v1", "deepseek-v4-flash"
	case "openai":
		return "https://api.openai.com/v1", "gpt-4o"
	case "ollama":
		return "http://localhost:11434/v1", "gemma4-hermes"
	default:
		return "", ""
	}
}
