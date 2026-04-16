package groq

import (
	openai "github.com/sashabaranov/go-openai"
)

var GroqClient *openai.Client

func Init(apiKey, baseURL string) {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	GroqClient = openai.NewClientWithConfig(cfg)
}
