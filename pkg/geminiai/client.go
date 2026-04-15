package geminiai

import (
	"context"

	"google.golang.org/genai"
)

type GeminiAIClient struct {
	client *genai.Client
}

var Genaiclient *genai.Client

func InitGeminiAIClient(apiKey string) error {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
	})
	if err != nil {
		return err
	}

	Genaiclient = client
	return nil
}
