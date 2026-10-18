// Package llm talks to any OpenAI-compatible chat completions endpoint
// (Groq, OpenRouter, Ollama, vLLM ...).
package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ravio1i/whatdowedo/internal/httpx"
)

// Client is a minimal chat-completions client.
type Client struct {
	BaseURL string
	APIKey  string
	Model   string
}

// Message is a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model          string         `json:"model"`
	Messages       []Message      `json:"messages"`
	Temperature    float64        `json:"temperature"`
	MaxTokens      int            `json:"max_tokens,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
}

type response struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Configured reports whether an API key is set. Ollama does not need one, so
// a base URL pointing at localhost also counts as configured.
func (c Client) Configured() bool {
	return c.APIKey != "" || strings.Contains(c.BaseURL, "localhost") || strings.Contains(c.BaseURL, "127.0.0.1")
}

// CompleteJSON asks the model for a JSON object and returns the raw content.
func (c Client) CompleteJSON(ctx context.Context, system, user string) (string, error) {
	if !c.Configured() {
		return "", errors.New("llm: LLM_API_KEY not configured")
	}
	req := request{
		Model:          c.Model,
		Messages:       []Message{{Role: "system", Content: system}, {Role: "user", Content: user}},
		Temperature:    0.7,
		MaxTokens:      2048,
		ResponseFormat: map[string]any{"type": "json_object"},
	}
	headers := map[string]string{}
	if c.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.APIKey
	}
	// OpenRouter attribution headers are harmless elsewhere.
	headers["HTTP-Referer"] = "https://github.com/ravio1i/whatdowedo"
	headers["X-Title"] = "whatdowedo"

	var resp response
	url := strings.TrimRight(c.BaseURL, "/") + "/chat/completions"
	if err := httpx.PostJSON(ctx, url, headers, req, &resp); err != nil {
		return "", fmt.Errorf("llm: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("llm: %s", resp.Error.Message)
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("llm: empty response")
	}
	return resp.Choices[0].Message.Content, nil
}
