package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"space-wars-3002-text-generation/internal/config"
)

type Client struct {
	baseURL string
	model   string
	client  *http.Client
	cfg     *config.Config
}

func New(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.LLMBaseURL,
		model:   cfg.LLMModel,
		client: &http.Client{
			Timeout: time.Duration(cfg.LLMTimeoutSeconds) * time.Second,
		},
		cfg: cfg,
	}
}

// Generate calls the LLM and returns the parsed lines.
// attempt=0 is the first try. attempt>0 lowers temperature and reinforces the plain-list constraint.
func (c *Client) Generate(system, user string, attempt int) ([]string, error) {
	temperature := c.cfg.LLMTemperature
	if attempt > 0 {
		// Lower temperature by 0.15 per retry, floor at 0.1
		temperature = temperature - float64(attempt)*0.15
		if temperature < 0.1 {
			temperature = 0.1
		}
		// Reinforce the plain-list constraint
		user = user + "\n\nIMPORTANT: Output the numbered list only. No preamble. No commentary. No JSON. Just lines numbered 1. 2. 3. etc."
	}

	req := ChatRequest{
		Model: c.model,
		Messages: []ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: temperature,
		TopP:        c.cfg.LLMTopP,
		MaxTokens:   c.cfg.LLMMaxTokens,
		Stream:      false,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/v1/chat/completions", c.baseURL),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM returned status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	content := chatResp.Choices[0].Message.Content

	lines := splitLines(content)
	if len(lines) == 0 {
		return nil, fmt.Errorf("LLM returned no lines")
	}

	return lines, nil
}

// splitLines splits a raw LLM response into individual dialogue lines.
// Blank lines and lines that are only whitespace are discarded.
func splitLines(content string) []string {
	raw := strings.Split(content, "\n")
	var lines []string
	for _, line := range raw {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
