package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestOpenAIStreamingChatCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected path /chat/completions, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "text/event-stream")

		chunks := []string{
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4","choices":[{"index":0,"delta":{"role":"assistant","content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-1","object":"chat.completion.chunk","model":"gpt-4","choices":[{"index":0,"delta":{"content":"!"},"finish_reason":"stop"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n\n"))
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	provider := NewOpenAIProvider(server.URL, "")

	req := &ChatCompletionRequest{
		Model: "gpt-4",
		Messages: []ChatMessage{
			{Role: "user", Content: "Hi"},
		},
	}

	var chunks []*StreamChunk
	err := provider.CreateChatCompletionStream(context.Background(), req, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		return nil
	})

	if err != nil {
		t.Fatalf("Streaming failed: %v", err)
	}

	if len(chunks) != 3 {
		t.Errorf("Expected 3 chunks, got %d", len(chunks))
	}

	var content strings.Builder
	for _, chunk := range chunks {
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
	}

	expected := "Hello there!"
	if content.String() != expected {
		t.Errorf("Expected content %q, got %q", expected, content.String())
	}

	if len(chunks) > 0 && len(chunks[len(chunks)-1].Choices) > 0 {
		finishReason := chunks[len(chunks)-1].Choices[0].FinishReason
		if finishReason != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %q", finishReason)
		}
	}
}

func TestMockProviderStreaming(t *testing.T) {
	provider := NewMockProvider()

	req := &ChatCompletionRequest{
		Model: "mock-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "test"},
		},
	}

	// Collect chunks
	var chunks []*StreamChunk
	var content strings.Builder

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := provider.CreateChatCompletionStream(ctx, req, func(chunk *StreamChunk) error {
		chunks = append(chunks, chunk)
		if len(chunk.Choices) > 0 {
			content.WriteString(chunk.Choices[0].Delta.Content)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Streaming failed: %v", err)
	}

	// Verify we got multiple chunks
	if len(chunks) < 2 {
		t.Errorf("Expected multiple chunks, got %d", len(chunks))
	}

	// Verify content contains mock prefix
	result := content.String()
	if !strings.Contains(result, "[mock streaming]") {
		t.Errorf("Expected content to contain '[mock streaming]', got %q", result)
	}

	// Verify first chunk has role
	if len(chunks) > 0 && len(chunks[0].Choices) > 0 {
		role := chunks[0].Choices[0].Delta.Role
		if role != "assistant" {
			t.Errorf("Expected first chunk role 'assistant', got %q", role)
		}
	}

	// Verify last chunk has finish reason
	if len(chunks) > 0 && len(chunks[len(chunks)-1].Choices) > 0 {
		finishReason := chunks[len(chunks)-1].Choices[0].FinishReason
		if finishReason != "stop" {
			t.Errorf("Expected finish_reason 'stop', got %q", finishReason)
		}
	}
}

func TestProviderStreamingInterface(t *testing.T) {
	// Verify all providers implement StreamingProtocol
	providers := []struct {
		name     string
		provider Protocol
	}{
		{"OpenAI", NewOpenAIProvider("http://test", "key")},
		{"Mock", NewMockProvider()},
	}

	for _, tc := range providers {
		t.Run(tc.name, func(t *testing.T) {
			_, ok := tc.provider.(StreamingProtocol)
			if !ok {
				t.Errorf("Provider %s does not implement StreamingProtocol", tc.name)
			}
		})
	}
}

func TestRegistryStreamingSupport(t *testing.T) {
	registry := NewRegistry()

	// Register providers
	providers := []struct {
		id       string
		typ      string
		endpoint string
	}{
		{"openai-test", "openai", "http://test"},
		{"local-test", "local", "http://test"},
		{"mock-test", "mock", ""},
	}

	for _, p := range providers {
		err := registry.Register(&ProviderConfig{
			ID:       p.id,
			Type:     p.typ,
			Endpoint: p.endpoint,
			Model:    "test-model",
		})
		if err != nil {
			t.Fatalf("Failed to register %s: %v", p.id, err)
		}
	}

	// Verify all support streaming via registry
	for _, p := range providers {
		registered, err := registry.Get(p.id)
		if err != nil {
			t.Fatalf("Failed to get provider %s: %v", p.id, err)
		}

		_, ok := registered.Protocol.(StreamingProtocol)
		if !ok {
			t.Errorf("Provider %s does not implement StreamingProtocol", p.id)
		}
	}
}
