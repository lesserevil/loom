package provider

import (
	"context"
	"time"
)

// MockProvider is an in-memory provider that returns canned responses.
// It is useful for local development and smoke-testing when no real model endpoint is available.
type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

// CreateChatCompletion returns a static echo response.
func (p *MockProvider) CreateChatCompletion(ctx context.Context, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	// Build a short echo message from the last user content.
	content := "mock response"
	if len(req.Messages) > 0 {
		content = req.Messages[len(req.Messages)-1].Content
		if content == "" {
			content = "mock response"
		}
	}

	resp := &ChatCompletionResponse{
		ID:      "mock-completion",
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []struct {
			Index   int         `json:"index"`
			Message ChatMessage `json:"message"`
			Finish  string      `json:"finish_reason"`
		}{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: "[mock] " + content,
				},
				Finish: "stop",
			},
		},
	}
	resp.Usage.PromptTokens = len(content)
	resp.Usage.CompletionTokens = len(resp.Choices[0].Message.Content)
	resp.Usage.TotalTokens = resp.Usage.PromptTokens + resp.Usage.CompletionTokens
	return resp, nil
}

// GetModels returns a single mock model.
func (p *MockProvider) GetModels(ctx context.Context) ([]Model, error) {
	return []Model{
		{
			ID:      "mock-model",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "mock",
		},
	}, nil
}
