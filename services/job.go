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
func (s *JobService) ProcessJob(ctx context.Context, title, description, siteUrl string, fileBytes []byte) error {
	bucket, err := beego.AppConfig.String("app::bucket_name")
	if err != nil || bucket == "" {
		return NewServiceError(ErrConfiguration, "BUCKET_NOT_CONFIGURED", "bucket name is not configured", err)
	}

	model, err := beego.AppConfig.String("app::groq_model")
	if err != nil || model == "" {
		return NewServiceError(ErrConfiguration, "LLM_MODEL_NOT_CONFIGURED", "llm model is not configured", err)
	}

	// Build and upload input JSON
	data, err := s.buildInputJSON(title, description, fileBytes)
	if err != nil {
		return err
	}

	storageService := NewStorageService(s3.S3Client)
	inputPath := storageService.BuildInputPath(siteUrl)

	if err := storageService.Upload(ctx, bucket, inputPath, data); err != nil {
		return NewServiceError(ErrStorage, "INPUT_UPLOAD_FAILED", "failed to upload input payload", err)
	}

	// Parse interpolated properties from input JSON
	var inputData struct {
		Properties []map[string]string `json:"properties"`
	}
	if err := json.Unmarshal(data, &inputData); err != nil {
		return NewServiceError(ErrSerialization, "INPUT_UNMARSHAL_FAILED", "failed to parse input payload", err)
	}
	if len(inputData.Properties) == 0 {
		return NewServiceError(ErrValidation, "NO_PROPERTIES_FOUND", "no properties found in csv", nil)
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

			rawOutput, err := llm.GenerateRawSEO(prop["title"], prop["description"])
			if err != nil {
				errCh <- NewServiceError(ErrLLM, "RAW_SEO_GENERATION_FAILED", fmt.Sprintf("raw seo generation failed for id=%s", prop["id"]), err)
				return
			}

			rawBytes, err := json.Marshal(rawOutput)
			if err != nil {
				errCh <- NewServiceError(ErrSerialization, "RAW_OUTPUT_MARSHAL_FAILED", "failed to encode raw llm output", err)
				return
			}
			outputPath := storageService.BuildOutputPath(siteUrl)

			var rawMap map[string]any
			if err := json.Unmarshal(rawBytes, &rawMap); err != nil {
				errCh <- NewServiceError(ErrSerialization, "RAW_OUTPUT_UNMARSHAL_FAILED", "failed to decode raw llm output", err)
				return
			}
			rawMap["id"] = prop["id"]
			rawBytes, err = json.Marshal(rawMap)
			if err != nil {
				errCh <- NewServiceError(ErrSerialization, "RAW_OUTPUT_REMARSHAL_FAILED", "failed to encode annotated raw llm output", err)
				return
			}

			if err := storageService.Upload(ctx, bucket, outputPath, rawBytes); err != nil {
				errCh <- NewServiceError(ErrStorage, "RAW_OUTPUT_UPLOAD_FAILED", "failed to upload raw llm output", err)
				return
			}

			seoResponse, err := llm.ProcessRawSEO(rawOutput)
			if err != nil {
				errCh <- NewServiceError(ErrProcessing, "RAW_SEO_PROCESSING_FAILED", fmt.Sprintf("processing raw seo failed for id=%s", prop["id"]), err)
				return
			}

			artifactData := map[string]string{
				"id":          prop["id"],
				"title":       seoResponse.Title,
				"description": seoResponse.Description,
			}
			artifactBytes, err := json.Marshal(artifactData)
			if err != nil {
				errCh <- NewServiceError(ErrSerialization, "ARTIFACT_MARSHAL_FAILED", "failed to encode artifact payload", err)
				return
			}
			artifactPath := storageService.BuildArtifactPath(siteUrl, prop["id"])
			if err := storageService.Upload(ctx, bucket, artifactPath, artifactBytes); err != nil {
				errCh <- NewServiceError(ErrStorage, "ARTIFACT_UPLOAD_FAILED", "failed to upload artifact payload", err)
			}
		}(prop)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *JobService) buildInputJSON(title, description string, file []byte) ([]byte, error) {
	properties := []map[string]string{}

	reader := csv.NewReader(bytes.NewReader(file))
	if _, err := reader.Read(); err != nil {
		return nil, NewServiceError(ErrProcessing, "CSV_HEADER_READ_FAILED", "failed to read csv header", err)
	}

	for {
		record, err := reader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, NewServiceError(ErrProcessing, "CSV_ROW_READ_FAILED", "failed to read csv row", err)
		}

		if len(record) < 3 {
			return nil, NewServiceError(ErrProcessing, "INVALID_CSV_ROW_LENGTH", "csv row must contain id,title,description", nil)
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

	data, err := json.Marshal(inputData)
	if err != nil {
		return nil, NewServiceError(ErrSerialization, "INPUT_MARSHAL_FAILED", "failed to encode input payload", err)
	}

	return data, nil
}

func ptr[T any](v T) *T {
	return &v
}
