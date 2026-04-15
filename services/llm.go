package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"google.golang.org/genai"
)

type SEOResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type LLMService struct {
	client *genai.Client
	model  string
}

func NewLLMService(client *genai.Client, model string) *LLMService {
	return &LLMService{
		client: client,
		model:  model,
	}
}

func (s *LLMService) GenerateSEO(title, description string) (SEOResponse, error) {
	ctx := context.Background()

	prompt := fmt.Sprintf(`
You are an expert travel blogger who creates content for high-performing travel accommodation, event, and activity booking websites that are search engine optimized. Rephrase the  "%s" title into a more SEO-friendly title. Use only the provided data without introducing new information or assumptions. The output should be in plain text. The text should be SEO-optimized for keywords related to the location. Write the sentences subtly, so that the keyword has the highest NLP Salience Score. Incorporate the location naturally within the content. Use pronouns or alternative references when the location has been established.
You are an expert travel blogger who creates content for high-performing travel accommodation, event and activity booking websites that are search engine optimized. Rephrase the following activity description "%s" into a more SEO-friendly and engaging paragraph by emphasizing key experiences, unique features, and specific location highlights to attract potential guests and improve online discoverability. Use only the content provided in the given description without adding new details or assumptions. The output should be in HTML markup. The text must contain exactly ONE HTML paragraph. Wrap the entire content inside a single <p>...</p> tag. Do not include any text or HTML elements outside this one <p> tag. The text should be SEO optimized towards the keywords: relating to the location. Write the sentences subtly, so that the keyword has the highest NLP Salience Score. Incorporate the location naturally within the content. Use pronouns or alternative references when the location has been established.
  `, title, description)
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseJsonSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]any{
					"type":        "string",
					"description": "SEO optimized title",
				},
				"description": map[string]any{
					"type":        "string",
					"description": "SEO optimized description wrapped in a single HTML paragraph tag",
				},
			},
			"required": []string{"title", "description"},
		},
	}

	result, err := s.client.Models.GenerateContent(
		ctx,
		s.model,
		genai.Text(prompt),
		config,
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result.Text())

	var seo SEOResponse
	err = json.Unmarshal([]byte(result.Text()), &seo)
	if err != nil {
		return SEOResponse{}, err
	}

	return seo, nil
}
