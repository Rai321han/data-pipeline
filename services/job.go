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

type jobConfig struct {
	bucket      string
	model       string
	temperature float32
	maxTokens   int
}

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
//   - An error if any step of the process fails, including configuration errors, storage errors, LLM processing errors, or serialization errors.
func (s *JobService) ProcessJob(ctx context.Context, title, description, siteUrl string, fileBytes []byte) error {
	config, err := s.loadJobConfig()
	if err != nil {
		return err
	}

	// Build and upload input JSON
	data, err := s.buildInputJSON(title, description, fileBytes)
	if err != nil {
		return err
	}

	storageService := NewStorageService(s3.S3Client)
	if err := s.uploadInputPayload(ctx, storageService, config.bucket, siteUrl, data); err != nil {
		return NewServiceError(ErrStorage, "INPUT_UPLOAD_FAILED", "failed to upload input payload", err)
	}

	properties, err := s.parseInputProperties(data)
	if err != nil {
		return err
	}

	llm := NewLLMService(groq.GroqClient, config.model, LLMConfig{
		Temperature:     config.temperature,
		MaxOutputTokens: config.maxTokens,
	})

	return s.processProperties(ctx, llm, storageService, config.bucket, siteUrl, properties)
}

// loadJobConfig reads the necessary configuration parameters for processing a job from the application configuration.
// It retrieves the S3 bucket name, LLM model name, temperature, and max output tokens from the configuration.
// If any required configuration is missing or invalid, it returns a ServiceError with appropriate context.
func (s *JobService) loadJobConfig() (jobConfig, error) {
	bucket, err := beego.AppConfig.String("app::bucket_name")
	if err != nil || bucket == "" {
		return jobConfig{}, NewServiceError(ErrConfiguration, "BUCKET_NOT_CONFIGURED", "bucket name is not configured", err)
	}

	model, err := beego.AppConfig.String("app::groq_model")
	if err != nil || model == "" {
		return jobConfig{}, NewServiceError(ErrConfiguration, "LLM_MODEL_NOT_CONFIGURED", "llm model is not configured", err)
	}

	temperature, err := beego.AppConfig.Float("app::temperature")
	if err != nil {
		temperature = 0.7
	}

	maxTokens, err := beego.AppConfig.Int("app::max_output_tokens")
	if err != nil {
		maxTokens = 1000
	}

	return jobConfig{
		bucket:      bucket,
		model:       model,
		temperature: float32(temperature),
		maxTokens:   maxTokens,
	}, nil
}

// uploadInputPayload uploads the input JSON payload to S3 at a path constructed using the site URL. It uses the provided storage service to perform the upload operation.
func (s *JobService) uploadInputPayload(ctx context.Context, storageService *storageService, bucket, siteURL string, data []byte) error {
	inputPath := storageService.BuildInputPath(siteURL)
	return storageService.Upload(ctx, bucket, inputPath, data)
}

func (s *JobService) parseInputProperties(data []byte) ([]map[string]string, error) {
	var inputData struct {
		Properties []map[string]string `json:"properties"`
	}

	if err := json.Unmarshal(data, &inputData); err != nil {
		return nil, NewServiceError(ErrSerialization, "INPUT_UNMARSHAL_FAILED", "failed to parse input payload", err)
	}

	if len(inputData.Properties) == 0 {
		return nil, NewServiceError(ErrValidation, "NO_PROPERTIES_FOUND", "no properties found in csv", nil)
	}

	return inputData.Properties, nil
}

// processProperties takes a list of properties extracted from the input JSON and processes each one concurrently.
func (s *JobService) processProperties(ctx context.Context, llm *LLMService, storageService *storageService, bucket, siteURL string, properties []map[string]string) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(properties))

	for _, prop := range properties {
		wg.Add(1)

		go func(property map[string]string) {
			defer wg.Done()
			if err := s.processProperty(ctx, llm, storageService, bucket, siteURL, property); err != nil {
				errCh <- err
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

// processProperty handles the processing of a single property, including generating raw SEO output using the LLM, uploading the raw output to S3, processing the raw output to produce clean SEO content, and uploading the final artifact back to S3.
func (s *JobService) processProperty(ctx context.Context, llm *LLMService, storageService *storageService, bucket, siteURL string, prop map[string]string) error {
	rawOutput, err := llm.GenerateRawSEO(prop["title"], prop["description"])
	if err != nil {
		return NewServiceError(ErrLLM, "RAW_SEO_GENERATION_FAILED", fmt.Sprintf("raw seo generation failed for id=%s", prop["id"]), err)
	}

	rawBytes, err := s.buildRawOutputPayload(rawOutput, prop["id"])
	if err != nil {
		return err
	}

	if err := storageService.Upload(ctx, bucket, storageService.BuildOutputPath(siteURL), rawBytes); err != nil {
		return NewServiceError(ErrStorage, "RAW_OUTPUT_UPLOAD_FAILED", "failed to upload raw llm output", err)
	}

	seoResponse, err := llm.ProcessRawSEO(rawOutput)
	if err != nil {
		return NewServiceError(ErrProcessing, "RAW_SEO_PROCESSING_FAILED", fmt.Sprintf("processing raw seo failed for id=%s", prop["id"]), err)
	}

	artifactBytes, err := s.buildArtifactPayload(prop["id"], seoResponse)
	if err != nil {
		return err
	}

	artifactPath := storageService.BuildArtifactPath(siteURL, prop["id"])
	if err := storageService.Upload(ctx, bucket, artifactPath, artifactBytes); err != nil {
		return NewServiceError(ErrStorage, "ARTIFACT_UPLOAD_FAILED", "failed to upload artifact payload", err)
	}

	return nil
}

// buildRawOutputPayload takes the raw output from the LLM and constructs a JSON payload that includes the original raw output along with the property ID.
// This payload is then uploaded to S3.
func (s *JobService) buildRawOutputPayload(rawOutput RawSEOOutput, propertyID string) ([]byte, error) {
	rawBytes, err := json.Marshal(rawOutput)
	if err != nil {
		return nil, NewServiceError(ErrSerialization, "RAW_OUTPUT_MARSHAL_FAILED", "failed to encode raw llm output", err)
	}

	var rawMap map[string]any
	if err := json.Unmarshal(rawBytes, &rawMap); err != nil {
		return nil, NewServiceError(ErrSerialization, "RAW_OUTPUT_UNMARSHAL_FAILED", "failed to decode raw llm output", err)
	}

	rawMap["id"] = propertyID
	rawBytes, err = json.Marshal(rawMap)
	if err != nil {
		return nil, NewServiceError(ErrSerialization, "RAW_OUTPUT_REMARSHAL_FAILED", "failed to encode annotated raw llm output", err)
	}

	return rawBytes, nil
}

// buildArtifactPayload takes the processed SEO response and constructs a JSON payload that includes the property ID, optimized title, and optimized description.
// This payload is then uploaded to S3 as the final artifact.
func (s *JobService) buildArtifactPayload(propertyID string, seoResponse SEOResponse) ([]byte, error) {
	artifactData := map[string]string{
		"id":          propertyID,
		"title":       seoResponse.Title,
		"description": seoResponse.Description,
	}

	artifactBytes, err := json.Marshal(artifactData)
	if err != nil {
		return nil, NewServiceError(ErrSerialization, "ARTIFACT_MARSHAL_FAILED", "failed to encode artifact payload", err)
	}

	return artifactBytes, nil
}

// buildInputJSON takes the original title and description prompts along with the CSV file bytes, parses the CSV to extract property data, and constructs a JSON payload that includes the prompts with placeholders replaced by actual property values.
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
