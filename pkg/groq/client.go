package groq

import (
	openai "github.com/sashabaranov/go-openai"
)

// GroqClient is a global variable that holds the initialized OpenAI client for making API calls to the Groq endpoint.
// It is initialized in the Init function with the provided API key and base URL.
var GroqClient *openai.Client

// Init initializes the GroqClient with the given API key and base URL. It should be called before making any API calls to ensure that the client is properly configured.
// Parameters:
//   - apiKey: The API key for authenticating with the OpenAI API.
//   - baseURL: The base URL for the OpenAI API, which can be customized for different environments (e.g., production, staging).
//
// This function sets up the GroqClient with the specified configuration, allowing it to be used throughout the application for generating SEO content using the LLM.
func Init(apiKey, baseURL string) {
	cfg := openai.DefaultConfig(apiKey)
	cfg.BaseURL = baseURL
	GroqClient = openai.NewClientWithConfig(cfg)
}
