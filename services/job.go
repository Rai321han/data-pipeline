package services

import (
	"bytes"
	"content_pipeline/pkg/groq"
	"content_pipeline/pkg/s3"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"

	beego "github.com/beego/beego/v2/server/web"
)

type JobService struct{}

// ProcessJob orchestrates the entire workflow of processing a job, including building the input JSON, uploading it to S3, generating SEO content using an LLM, and saving the results back to S3.
//
// Parameters:
//
//   - ctx: The context for managing request-scoped values, cancellation signals, and deadlines across API boundaries.
//
//   - title: A string representing the prompt for optimizing a title. This prompt may contain placeholders that will be replaced with actual property data from the CSV file.
//
//   - description: A string representing the prompt for optimizing a description. Similar to the title prompt, it may contain placeholders for dynamic content.
//
//   - siteUrl: A string representing the URL of the site being processed. This is used to construct S3 paths for storing input and output data.
//
//   - fileBytes: A byte slice containing the contents of the uploaded CSV file. The CSV file must have a header row with the columns "id", "title", and "description", and at least one data row. Each data row must have non-empty values for all three columns, and the "id" values must be unique.
//
// Returns:
//
//   - A string representing the S3 path where the input JSON was uploaded.
//
//   - An error object if any step of the process fails, or nil if the entire workflow completes successfully.
func (s *JobService) ProcessJob(ctx context.Context, title, description, siteUrl string, fileBytes []byte) (string, error) {
	bucket, err := beego.AppConfig.String("app::bucket_name")
	if err != nil || bucket == "" {
		return "", errors.New("bucket name is not configured")
	}

	model, err := beego.AppConfig.String("app::groq_model")
	if err != nil || model == "" {
		return "", errors.New("llm model is not configured")
	}

	// Build and upload input JSON
	data, err := s.buildInputJSON(title, description, fileBytes)
	if err != nil {
		return "", err
	}

	storageService := NewStorageService(s3.S3Client)
	inputPath := storageService.BuildInputPath(siteUrl)

	if err := storageService.Upload(ctx, bucket, inputPath, data); err != nil {
		return "", err
	}

	// Parse interpolated properties from input JSON
	var inputData struct {
		Properties []map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(data, &inputData); err != nil {
		return "", err
	}
	if len(inputData.Properties) == 0 {
		return "", errors.New("no properties found in CSV")
	}

	temperate, err := beego.AppConfig.Float("app::temperature")
	if err != nil {
		temperate = 0.7
	}

	maxTokens, err := beego.AppConfig.Int("app::max_output_tokens")
	if err != nil {
		maxTokens = 1000
	}

	llm := NewLLMService(groq.GroqClient, model, LLMConfig{
		Temperature:     float32(temperate),
		MaxOutputTokens: maxTokens,
	})

	var wg sync.WaitGroup
	errCh := make(chan error, len(inputData.Properties))

	for _, prop := range inputData.Properties {
		wg.Add(1)

		go func(prop map[string]string) {
			defer wg.Done()

			// Generate SEO content using the interpolated prompts
			rawOutput, err := llm.GenerateRawSEO(prop["title"], prop["description"])
			if err != nil {
				errCh <- fmt.Errorf("raw SEO generation failed for %s: %w", prop["id"], err)
				return
			}

			rawBytes, err := json.Marshal(rawOutput)
			if err != nil {
				errCh <- err
				return
			}
			outputPath := storageService.BuildOutputPath(siteUrl)

			// add id to the rawoutput for easier debugging
			var rawMap map[string]any
			if err := json.Unmarshal(rawBytes, &rawMap); err != nil {
				errCh <- err
				return
			}
			rawMap["id"] = prop["id"]
			rawBytes, err = json.Marshal(rawMap)
			if err != nil {
				errCh <- err
				return
			}

			// Step 1: save raw LLM output for debugging
			if err := storageService.Upload(ctx, bucket, outputPath, rawBytes); err != nil {
				errCh <- err
				return
			}

			// Step 2: process raw output into clean SEO content
			seoResponse, err := llm.ProcessRawSEO(rawOutput)
			if err != nil {
				errCh <- fmt.Errorf("processing raw SEO failed for %s: %w", prop["id"], err)
				return
			}

			// Step 3: save artifact
			artifactData := map[string]string{
				"id":          prop["id"],
				"title":       seoResponse.Title,
				"description": seoResponse.Description,
			}
			artifactBytes, err := json.Marshal(artifactData)
			if err != nil {
				errCh <- err
				return
			}
			artifactPath := storageService.BuildArtifactPath(siteUrl, prop["id"])
			if err := storageService.Upload(ctx, bucket, artifactPath, artifactBytes); err != nil {
				errCh <- err
			}
		}(prop)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return "", err
		}
	}

	return inputPath, nil
}

func (s *JobService) buildInputJSON(title, description string, file []byte) ([]byte, error) {
	properties := []map[string]string{}

	reader := csv.NewReader(bytes.NewReader(file))
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		propertyID := record[0]
		propertyTitle := record[1]
		propertyDescription := record[2]

		titlePrompt := strings.ReplaceAll(title, "{PropertyName}", propertyTitle)
		descriptionPrompt := strings.ReplaceAll(description, "{PropertyDescription}", propertyDescription)

		properties = append(properties, map[string]string{
			"id":          propertyID,
			"title":       titlePrompt,
			"description": descriptionPrompt,
		})
	}

	inputData := map[string]any{
		"properties": properties,
	}

	return json.Marshal(inputData)
}

func ptr[T any](v T) *T {
	return &v
}
