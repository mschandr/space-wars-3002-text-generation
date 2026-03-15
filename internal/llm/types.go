package llm

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	TopP        float64       `json:"top_p"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type Choice struct {
	Message ChatMessage `json:"message"`
}

type ChatResponse struct {
	Choices []Choice `json:"choices"`
}

type DialogueOutput struct {
	Lines []string `json:"lines"`
}
