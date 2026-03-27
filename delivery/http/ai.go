package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// generateInsight sends the prompt to a generative API and returns a short text insight.
// Works with Groq APIs.
func generateInsight(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", errors.New("GEMINI_API_KEY not set")
	}

	apiURL := os.Getenv("GEMINI_API_URL")
	if apiURL == "" {
		apiURL = "https://api.groq.com/openai/v1/chat/completions"
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	// OpenAI-compatible request format (works with Groq)
	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]interface{}{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  200,
		"temperature": 0.3,
	}

	b, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	// For OpenAI-compatible APIs, use Bearer token in header
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, strings.NewReader(string(b)))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyStr := strings.TrimSpace(string(respBody))
		if bodyStr == "" {
			return "", fmt.Errorf("ai service returned status %s", resp.Status)
		}
		return "", fmt.Errorf("ai service returned status %s: %s", resp.Status, bodyStr)
	}

	// Parse OpenAI-compatible response
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		// Fallback: return raw response if parsing fails
		return strings.TrimSpace(string(respBody)), nil
	}

	if len(openAIResp.Choices) == 0 {
		return "", errors.New("no choices in response")
	}

	insight := strings.TrimSpace(openAIResp.Choices[0].Message.Content)
	if insight == "" {
		return "", errors.New("empty insight from AI")
	}

	return insight, nil
}
