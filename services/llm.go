package services

import (
	"context"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type SEOResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type RawSEOOutput struct {
	RawTitle       string `json:"raw_title"`
	RawDescription string `json:"raw_description"`
}

type LLMConfig struct {
	Temperature     float32
	MaxOutputTokens int
	TopP            float32
}

type LLMService struct {
	client *openai.Client
	model  string
	config LLMConfig
}

func NewLLMService(client *openai.Client, model string, cfg LLMConfig) *LLMService {
	return &LLMService{
		client: client,
		model:  model,
		config: cfg,
	}
}

func (s *LLMService) GenerateRawSEO(titlePrompt, descriptionPrompt string) (RawSEOOutput, error) {
	ctx := context.Background()

	rawTitle, err := s.generate(ctx, titlePrompt)
	if err != nil {
		return RawSEOOutput{}, NewServiceError(ErrLLM, "TITLE_GENERATION_FAILED", "title generation failed", err)
	}

	rawDescription, err := s.generate(ctx, descriptionPrompt)
	if err != nil {
		return RawSEOOutput{}, NewServiceError(ErrLLM, "DESCRIPTION_GENERATION_FAILED", "description generation failed", err)
	}

	return RawSEOOutput{
		RawTitle:       rawTitle,
		RawDescription: rawDescription,
	}, nil
}

func (s *LLMService) generate(ctx context.Context, prompt string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: s.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
		Temperature: s.config.Temperature,
		MaxTokens:   s.config.MaxOutputTokens,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", NewServiceError(ErrLLM, "LLM_REQUEST_FAILED", "llm request failed", err)
	}

	if len(resp.Choices) == 0 {
		return "", NewServiceError(ErrLLM, "EMPTY_LLM_RESPONSE", "no choices returned from model", nil)
	}

	return resp.Choices[0].Message.Content, nil
}

func (s *LLMService) ProcessRawSEO(raw RawSEOOutput) (SEOResponse, error) {
	title := stripHTML(strings.TrimSpace(raw.RawTitle))
	description := normalizeParagraph(strings.TrimSpace(raw.RawDescription))

	if title == "" {
		return SEOResponse{}, NewServiceError(ErrProcessing, "EMPTY_PROCESSED_TITLE", "processed title is empty", nil)
	}
	if description == "" {
		return SEOResponse{}, NewServiceError(ErrProcessing, "EMPTY_PROCESSED_DESCRIPTION", "processed description is empty", nil)
	}

	return SEOResponse{
		Title:       title,
		Description: description,
	}, nil
}

func stripHTML(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}

func normalizeParagraph(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "<p>") && strings.HasSuffix(s, "</p>") {
		s = s[3 : len(s)-4]
	}
	return "<p>" + strings.TrimSpace(s) + "</p>"
}
